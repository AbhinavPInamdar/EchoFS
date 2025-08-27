package fileops

type ChunkMeta struct {
	FileName string `json:"filename"`
	MD5Hash  string `json:"md5_hash"`
	Index    int    `json:"index"`
}

type Config struct {
	ChunkSize int
	ServerURL string //is this needed? check later
}

type DefaultFileChunker struct {
	chunkSize int
}

func NewDefaultFileChunker(chunkSize int) *DefaultFileChunker {
	return &DefaultFileChunker{
		chunkSize: chunkSize,
	}
}

type DefaultFileUploader struct {
	serverURL string
}

type FileChunker interface {
	ChunkFile(filePath string) ([]ChunkMeta, error)
	ChunkLargeFile(filePath string) ([]ChunkMeta, error)
}

type Uploader interface {
	UploadChunk(chunk ChunkMeta) error
}

type MetadataManager interface {
	LoadMetadata(filePath string) (map[string]ChunkMeta, error)
	SaveMetadata(filePath string, metadata map[string]ChunkMeta) error
}
