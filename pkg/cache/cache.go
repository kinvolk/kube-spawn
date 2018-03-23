package cache

type Cache struct {
	dir string
}

func New(dir string) (*Cache, error) {
	return &Cache{
		dir: dir,
	}, nil
}

func (c *Cache) Dir() string {
	return c.dir
}
