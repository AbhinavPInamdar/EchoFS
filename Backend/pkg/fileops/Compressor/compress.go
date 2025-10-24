package compressor

import (
	"compress/gzip"
	"io"
	"os"
)

func Compress(filePath string) (*os.File, error) {
	inputFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer inputFile.Close()

	outputFileName := filePath + ".gz"
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		return nil, err
	}

	gzipWriter := gzip.NewWriter(outputFile)
	_, err = io.Copy(gzipWriter, inputFile)
	if err != nil {
		outputFile.Close()
		return nil, err
	}

	gzipWriter.Close()
	outputFile.Seek(0, 0)
	return outputFile, nil
}