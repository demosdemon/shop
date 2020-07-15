package config

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/demosdemon/shop/pkg/secrets"
	"github.com/demosdemon/shop/pkg/secrets/awsparamstore"
)

var (
	ErrNoRepository = errors.New("no repository specified")
)

var pathRegexp = regexp.MustCompile(`^(?:[^/]+)/(?:.+?).ya?ml$`)

type Error struct {
	Step   string
	Repo   string
	Branch string
	Commit string
	File   string
	Err    error
}

func (e Error) Error() string {
	return fmt.Sprintf(
		"error processing step:%q repo:%q branch:%q commit:%q file:%q error:%v",
		e.Step,
		e.Repo,
		e.Branch,
		e.Commit,
		e.File,
		e.Err,
	)
}

func (r *Runtime) ScanRepository() ([]*Store, error) {
	if r.RepositoryPath == "" {
		return nil, ErrNoRepository
	}

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	creds, err := sess.Config.Credentials.Get()
	if err != nil {
		return nil, err
	}

	if !creds.HasKeys() {
		return nil, errors.New("invalid AWS credentials")
	}

	resolver := awsparamstore.New(sess)

	fs := osfs.New(r.RepositoryPath)
	if _, err := fs.Stat(git.GitDirName); err == nil {
		fs, err = fs.Chroot(git.GitDirName)
		if err != nil {
			return nil, Error{
				Step: "opening storage for repo",
				Repo: r.RepositoryPath,
				Err:  err,
			}
		}
	}

	s := filesystem.NewStorageWithOptions(
		fs,
		cache.NewObjectLRUDefault(),
		filesystem.Options{},
	)
	defer func() { _ = s.Close() }()

	repo, err := git.Open(s, fs)
	if err != nil {
		return nil, Error{
			Step: "opening git repository",
			Repo: r.RepositoryPath,
			Err:  err,
		}
	}

	iter, err := repo.Branches()
	if err != nil {
		return nil, Error{
			Step: "enumerating branches",
			Repo: r.RepositoryPath,
			Err:  err,
		}
	}

	storesMap := make(map[string]*Store)
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		hash := ref.Hash()
		commit, err := repo.CommitObject(hash)
		if err != nil {
			return Error{
				Step:   "opening commit blob",
				Repo:   r.RepositoryPath,
				Commit: hash.String(),
				Branch: ref.Name().Short(),
				Err:    err,
			}
		}
		tree, err := commit.Tree()
		if err != nil {
			return Error{
				Step:   "opening tree blob",
				Repo:   r.RepositoryPath,
				Commit: hash.String(),
				Branch: ref.Name().Short(),
				Err:    err,
			}
		}

		tree, err = tree.Tree("etc")
		if err != nil {
			if err == object.ErrDirectoryNotFound {
				return nil
			}

			return Error{
				Step:   "opening etc tree blob",
				Repo:   r.RepositoryPath,
				Commit: hash.String(),
				Branch: ref.Name().Short(),
				Err:    err,
			}
		}

		return tree.Files().ForEach(func(file *object.File) error {
			if pathRegexp.MatchString(file.Name) {
				fp, err := file.Reader()
				if err != nil {
					return Error{
						Step:   "opening file blob",
						Repo:   r.RepositoryPath,
						Commit: hash.String(),
						Branch: ref.Name().Short(),
						File:   file.Name,
						Err:    err,
					}
				}

				s, err := ReadStores(file.Name, hash.String(), fp, resolver)
				if err != nil {
					err = Error{
						Step:   "reading file",
						Repo:   r.RepositoryPath,
						Commit: hash.String(),
						Branch: ref.Name().Short(),
						File:   file.Name,
						Err:    err,
					}
				}
				if err2 := fp.Close(); err == nil && err2 != nil {
					err = Error{
						Step:   "closing file after reading stores",
						Repo:   r.RepositoryPath,
						Commit: hash.String(),
						Branch: ref.Name().Short(),
						File:   file.Name,
						Err:    err2,
					}
				}

				for _, s := range s {
					storesMap[s.StoreID] = s
				}

				return err
			}

			return nil
		})
	})

	stores := make([]*Store, 0, len(storesMap))
	for _, store := range storesMap {
		stores = append(stores, store)
	}

	sort.Slice(stores, func(i, j int) bool { return stores[i].StoreID < stores[j].StoreID })

	return stores, err
}

func ReadStores(path, hash string, r io.Reader, resolver secrets.Resolver) ([]*Store, error) {
	ctx := context.Background()

	var config struct {
		Integrations []struct {
			Id       *string         `yaml:"id"`
			Strategy string          `yaml:"strategy"`
			Sync     *bool           `yaml:"sync"`
			Enabled  *bool           `yaml:"enabled"`
			Options  secrets.Secrets `yaml:"options"`
		} `yaml:"integrations"`
	}

	dec := yaml.NewDecoder(r)
	if err := dec.Decode(&config); err != nil {
		return nil, err
	}

	stores := make([]*Store, 0, len(config.Integrations))
	for _, integration := range config.Integrations {
		if integration.Strategy == "shopify" && (integration.Enabled == nil || *integration.Enabled) && (integration.Sync == nil || *integration.Sync) {
			id := integration.Strategy
			if integration.Id != nil {
				id = *integration.Id
			}

			storeId, err := oneOf(ctx, resolver, integration.Options.Get("name"), integration.Options.Get("id"))
			if err != nil {
				return nil, err
			}

			username, err := get(ctx, resolver, integration.Options.Get("key"))
			if err != nil {
				return nil, err
			}

			password, err := get(ctx, resolver, integration.Options.Get("password"))
			if err != nil {
				return nil, err
			}

			store := &Store{
				File:     path,
				ID:       id,
				StoreID:  storeId,
				Username: username,
				Password: password,
				Commit:   hash,
			}

			stores = append(stores, store)
		}
	}

	return stores, nil
}

func oneOf(ctx context.Context, resolver secrets.Resolver, a, b *secrets.Secret) (v string, err error) {
	v, err = get(ctx, resolver, a)

	if v == "" || err != nil {
		v, err = get(ctx, resolver, b)
	}

	if v == "" && err != nil {
		err = errors.New("invalid secret")
	}

	return
}

func get(ctx context.Context, resolver secrets.Resolver, s *secrets.Secret) (v string, err error) {
	if s == nil {
		err = errors.New("nil secret")
		return
	}

	return s.Resolve(ctx, resolver)
}
