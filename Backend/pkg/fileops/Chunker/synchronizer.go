package fileops

import (
	"sync"
)

func synchronizerChunks(chunks []ChunkMeta, metadata map[string]ChunkMeta, upload Uploader, wg *sync.WaitGroup, mu *sync.Mutex) error {
	chunkChannel := make(chan ChunkMeta, len(chunks))
	errChannel := make(chan error, len(chunks))

	for _, chunk := range chunks {
		chunkChannel <- chunk
	}
	close(chunkChannel)

	numWorkers := 4
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for chunk := range chunkChannel {
				mu.Lock()
				oldChunk, exists := metadata[chunk.FileName]
				mu.Unlock()

				if !exists || oldChunk.MD5Hash != chunk.MD5Hash {
					if err := upload.UploadChunk(chunk); err != nil {
						errChannel <- err
						return
					}

					mu.Lock()
					metadata[chunk.FileName] = chunk
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
	close(errChannel)

	for err := range errChannel {
		if err != nil {
			return err
		}
	}
	return nil
}
