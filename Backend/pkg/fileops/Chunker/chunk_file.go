package fileops

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"
)

func (c *DefaultFileChunker) ChunkFile(filePath string) ([]ChunkMeta, error) {
	var chunks []ChunkMeta

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buffer := make([]byte, c.chunkSize)
	index := 0
	for {
		bytesread, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if bytesread == 0 {
			break
		}
		hash := md5.Sum(buffer[:bytesread])
		hashString := hex.EncodeToString(hash[:])
		chunkFileName := fmt.Sprintf("%s.chunk.%d", filePath, index)
		chunkFile, err := os.Create(chunkFileName)
		if err != nil {
			return nil, err
		}
		_, err = chunkFile.Write(buffer[:bytesread])
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, ChunkMeta{FileName: chunkFileName, MD5Hash: hashString, Index: index})
		chunkFile.Close()
		index++
	}
	return chunks, nil
}

func (c *DefaultFileChunker) ChunkLargeFile(filePath string) ([]ChunkMeta, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var chunks []ChunkMeta
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	numChunks := int(fileInfo.Size() / int64(c.chunkSize))
	if fileInfo.Size()%int64(c.chunkSize) != 0 {
		numChunks++
	}
	chunkChannel := make(chan ChunkMeta, numChunks)
	errChannel := make(chan error, numChunks)
	indexChannel := make(chan int, numChunks)

	for i := 0; i < numChunks; i++ {
		indexChannel <- i
	}
	close(indexChannel)

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range indexChannel {
				offset := int64(index) * int64(c.chunkSize)
				buffer := make([]byte, c.chunkSize)

				file.Seek(offset, 0)

				bytesRead, err := file.Read(buffer)
				if err != nil && err != io.EOF {
					errChannel <- err
					return
				}
				if bytesRead > 0 {
					hash := md5.Sum(buffer[:bytesRead])
					hashString := hex.EncodeToString(hash[:])

					chunkFileName := fmt.Sprintf("%s.chunk.%d", filePath, index)

					chunkFile, err := os.Create(chunkFileName)
					if err != nil {
						errChannel <- err
						return
					}

					_, err = chunkFile.Write(buffer[:bytesRead])
					if err != nil {
						errChannel <- err
						return
					}
					chunk := ChunkMeta{
						FileName: chunkFileName,
						MD5Hash:  hashString,
						Index:    index,
					}
					mu.Lock()
					chunks = append(chunks, chunk)
					mu.Unlock()

					chunkFile.Close()

					chunkChannel <- chunk
				}

			}

		}()
	}

	go func() {
		wg.Wait()
		close(chunkChannel)
		close(errChannel)
	}()

	for err := range errChannel {
		if err != nil {
			return nil, err
		}
	}
	return chunks, nil
}
