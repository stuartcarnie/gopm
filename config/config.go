package config

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"

	"github.com/hashicorp/go-memdb"
	"github.com/r3labs/diff"
)

// Config memory representations of supervisor configuration file
type Config struct {
	db *memdb.MemDB
}

// NewConfig returns a new Config value.
func NewConfig() *Config {
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"process": {
				Name: "process",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Name"},
					},
				},
			},
			"server": {
				Name: "server",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Name"},
					},
				},
			},
			"file": {
				Name: "file",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Name"},
					},
				},
			},
			"local_file": {
				Name: "local_file",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Name"},
					},
				},
			},
		},
	}

	db, err := memdb.NewMemDB(schema)
	if err != nil {
		panic(err)
	}

	return &Config{
		db: db,
	}
}

// Load loads the configuration and return the loaded programs
func (c *Config) LoadPath(configFile string) (memdb.Changes, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	m, err := parseRoot(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return c.update(m)
}

func (c *Config) update(m *root) (memdb.Changes, error) {

	txn := c.db.Txn(true)
	txn.TrackChanges()
	err := applyUpdates(txn, m)
	if err != nil {
		txn.Abort()
		return nil, err
	}

	ri, err := txn.Get("file", "id")
	if err != nil {
		panic(err)
	}

	var files []*file
	for {
		f, ok := ri.Next().(*file)
		if !ok {
			break
		}
		files = append(files, f)
	}

	// update local files table
	if len(files) > 0 {
		root := m.Runtime.Root
		localFiles, err := writeFiles(root, files)
		if err != nil {
			txn.Abort()
			return nil, err
		}

		_ = os.Setenv("GOPM_FS_ROOT", root)

		for _, lf := range localFiles {
			raw, _ := txn.First("local_file", "id", lf.name)
			if orig, ok := raw.(*localFile); ok && !diff.Changed(orig, lf) {
				continue
			}
			_ = txn.Insert("local_file", lf)
			// TODO(sgc): Want to merge these with each process environment
			key := strings.ToUpper("GOPM_FS_" + lf.name)
			_ = os.Setenv(key, lf.fullPath)
		}
	}

	ch := txn.Changes()
	txn.Commit()

	return ch, nil
}

func (c *Config) GetGrpcServer() *Server {
	txn := c.db.Txn(false)
	defer txn.Commit()
	raw, err := txn.First("server", "id", "grpc")
	if raw == nil || err != nil {
		return nil
	}
	return raw.(*Server)
}
