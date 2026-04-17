package platform

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"regexp"
)

// LogIndex 日志索引结构
type LogIndex struct {
	Version     uint32
	FilePath    string
	FileSize    int64
	TotalLines  int64
	FrameStarts []int64 // 每帧在文件中的偏移量
}

// StreamReader 大文件流式读取器
type StreamReader struct {
	file     *os.File
	reader   *bufio.Reader
	fileSize int64
	filePath string
}

// BOM signatures
var (
	UTF8BOM    = []byte{0xEF, 0xBB, 0xBF}
	UTF16LEBOM = []byte{0xFF, 0xFE}
	UTF16BEBOM = []byte{0xFE, 0xFF}
)

// DetectEncoding 检测文件编码
func DetectEncoding(firstBytes []byte) string {
	if len(firstBytes) < 3 {
		return "utf-8"
	}
	if bytes.HasPrefix(firstBytes, UTF8BOM) {
		return "utf-8"
	}
	if bytes.HasPrefix(firstBytes, UTF16LEBOM) {
		return "utf-16-le"
	}
	if bytes.HasPrefix(firstBytes, UTF16BEBOM) {
		return "utf-16-be"
	}
	// For logs without BOM, default to UTF-8
	// Note: GBK support was removed as convertLine does not implement it
	return "utf-8"
}

// NewStreamReader 创建流式读取器
func NewStreamReader(filePath string) (*StreamReader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	// Read first bytes for BOM detection
	firstBytes := make([]byte, 64)
	n, err := file.Read(firstBytes)
	if err != nil {
		file.Close()
		return nil, err
	}
	if n > 0 {
		// Seek back to beginning
		file.Seek(0, io.SeekStart)
	}

	sr := &StreamReader{
		file:     file,
		reader:   bufio.NewReader(file),
		fileSize: stat.Size(),
		filePath: filePath,
	}

	return sr, nil
}

// GetFileSize 返回文件大小
func (sr *StreamReader) GetFileSize() int64 {
	return sr.fileSize
}

// ReadLines 返回行读取迭代器
func (sr *StreamReader) ReadLines() <-chan string {
	return sr.ReadLinesWithProgress(nil)
}

// ReadLinesWithProgress 带进度回调的行读取
func (sr *StreamReader) ReadLinesWithProgress(onProgress func(float64)) <-chan string {
	ch := make(chan string, 1000)
	encoding := sr.detectEncoding()

	go func() {
		defer close(ch)

		lineNum := 0
		scanner := bufio.NewScanner(sr.reader)

		// Set larger buffer for large log lines (64MB max)
		maxCapacity := 64 * 1024 * 1024
		buf := make([]byte, maxCapacity)
		scanner.Buffer(buf, maxCapacity)

		readBytes := int64(0)
		for scanner.Scan() {
			lineNum++
			line := scanner.Bytes()

			// Convert encoding if needed
			text := sr.convertLine(line, encoding)
			readBytes += int64(len(line))

			if onProgress != nil && sr.fileSize > 0 {
				progress := float64(readBytes) / float64(sr.fileSize) * 100
				onProgress(progress)
			}

			ch <- text
		}
	}()

	return ch
}

// detectEncoding 检测文件编码
func (sr *StreamReader) detectEncoding() string {
	firstBytes := make([]byte, 64)
	n, err := sr.file.Read(firstBytes)
	if err != nil {
		return "utf-8"
	}
	sr.file.Seek(0, io.SeekStart)
	return DetectEncoding(firstBytes[:n])
}

// convertLine 转换行编码
func (sr *StreamReader) convertLine(line []byte, encoding string) string {
	// Currently only UTF-8 is supported. GBK detection was removed since
	// encoding conversion was not implemented.
	return string(line)
}

// SeekToLine 跳转到指定行（通过重新打开文件并跳过行）
func (sr *StreamReader) SeekToLine(lineNum int) error {
	sr.file.Seek(0, io.SeekStart)
	sr.reader = bufio.NewReader(sr.file)

	lineCount := 0
	scanner := bufio.NewScanner(sr.reader)

	maxCapacity := 64 * 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		lineCount++
		if lineCount >= lineNum {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

// Close 关闭文件
func (sr *StreamReader) Close() error {
	if sr.file != nil {
		return sr.file.Close()
	}
	return nil
}

// Index file format constants
const (
	IndexVersion     uint32 = 1
	IndexMagicNumber uint32 = 0x47535449 // "GSTI" in ASCII
)

// WriteIndex 写入索引文件
func WriteIndex(filePath string, index *LogIndex) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write magic number
	if err := binary.Write(f, binary.LittleEndian, IndexMagicNumber); err != nil {
		return err
	}

	// Write version
	if err := binary.Write(f, binary.LittleEndian, IndexVersion); err != nil {
		return err
	}

	// Write file path length and path
	pathLen := uint32(len(index.FilePath))
	if err := binary.Write(f, binary.LittleEndian, pathLen); err != nil {
		return err
	}
	if _, err := f.WriteString(index.FilePath); err != nil {
		return err
	}

	// Write file size
	if err := binary.Write(f, binary.LittleEndian, index.FileSize); err != nil {
		return err
	}

	// Write total lines
	if err := binary.Write(f, binary.LittleEndian, index.TotalLines); err != nil {
		return err
	}

	// Write frame starts count and data
	frameCount := uint32(len(index.FrameStarts))
	if err := binary.Write(f, binary.LittleEndian, frameCount); err != nil {
		return err
	}
	for _, offset := range index.FrameStarts {
		if err := binary.Write(f, binary.LittleEndian, offset); err != nil {
			return err
		}
	}

	return nil
}

// ReadIndex 读取索引文件
func ReadIndex(filePath string) (*LogIndex, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var magic uint32
	if err := binary.Read(f, binary.LittleEndian, &magic); err != nil {
		return nil, err
	}
	if magic != IndexMagicNumber {
		return nil, errors.New("invalid index file format")
	}

	var version uint32
	if err := binary.Read(f, binary.LittleEndian, &version); err != nil {
		return nil, err
	}
	if version != IndexVersion {
		return nil, errors.New("unsupported index version")
	}

	var pathLen uint32
	if err := binary.Read(f, binary.LittleEndian, &pathLen); err != nil {
		return nil, err
	}

	pathBytes := make([]byte, pathLen)
	if _, err := io.ReadFull(f, pathBytes); err != nil {
		return nil, err
	}

	index := &LogIndex{
		FilePath: string(pathBytes),
	}

	if err := binary.Read(f, binary.LittleEndian, &index.FileSize); err != nil {
		return nil, err
	}

	if err := binary.Read(f, binary.LittleEndian, &index.TotalLines); err != nil {
		return nil, err
	}

	var frameCount uint32
	if err := binary.Read(f, binary.LittleEndian, &frameCount); err != nil {
		return nil, err
	}

	index.FrameStarts = make([]int64, frameCount)
	for i := uint32(0); i < frameCount; i++ {
		if err := binary.Read(f, binary.LittleEndian, &index.FrameStarts[i]); err != nil {
			return nil, err
		}
	}

	return index, nil
}

// SearchPattern 在文件中搜索正则表达式匹配的行
func SearchPattern(filePath string, pattern string) ([]int64, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	sr, err := NewStreamReader(filePath)
	if err != nil {
		return nil, err
	}
	if sr != nil {
		defer sr.Close()
	}

	var lineNum int64
	var positions []int64

	for line := range sr.ReadLines() {
		if re.MatchString(line) {
			positions = append(positions, lineNum)
		}
		lineNum++
	}

	return positions, nil
}
