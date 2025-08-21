package fileops

import (
	"sync"
)

func synchronizerChunks(chunks []ChunkMeta, metadata map[string]ChunkMeta, upload Uploader, wg *sync.WaitGroup, mu *sync.Mutex) error {
	chunkChannel := make(chan ChunkMeta, len(chunks))
	errChannel := make(chan error, len(chunks))

	// enqueue chunks
	for _, chunk := range chunks {
		chunkChannel <- chunk
	}
	close(chunkChannel)

	// worker pool
	numWorkers := 4
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for chunk := range chunkChannel {
				mu.Lock()
				oldChunk, exists := metadata[chunk.FileName]
				mu.Unlock()

				// Only upload if new or changed
				if !exists || oldChunk.MD5Hash != chunk.MD5Hash {
					if err := upload.UploadChunk(chunk); err != nil {
						errChannel <- err
						return
					}

					// Update metadata
					mu.Lock()
					metadata[chunk.FileName] = chunk
					mu.Unlock()
				}
			}
		}()
	}

	// wait for workers
	wg.Wait()
	close(errChannel)

	// check for errors
	for err := range errChannel {
		if err != nil {
			return err
		}
	}
	return nil
}
