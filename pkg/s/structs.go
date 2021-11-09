package s

type Cache struct {
	Scope           string `json:"scope"`
	CacheKey        string `json:"cacheKey"`
	CacheVersion    string `json:"cacheVersion"`
	CreationTime    string `json:"creationTime"` // 2021-11-02T23:02:58.89Z
	ArchiveLocation string `json:"archiveLocation"`

	// Used to store path info so the archive url can be calculated
	StorageBackendType string `json:"-"`
	StorageBackendPath string `json:"-"`
}

type Scope struct {
	Scope      string `json:"Scope"`
	Permission int    `json:"Permission"`
}

type CachePart struct {
	Start int
	End   int
	Size  int64
	Data  string
}
