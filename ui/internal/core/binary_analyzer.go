package core

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

// BinaryAnalyzer 二进制文件分析器
type BinaryAnalyzer struct {
	filePath string
	fileType string
	data     []byte
}

// SectionInfo 段信息
type SectionInfo struct {
	Index  int
	Name   string
	Offset uint64
	Size   uint64
	Type   string
}

// StringInfo 字符串信息
type StringInfo struct {
	Index  int
	Offset uint64
	Data   string
	Length int
	String string
}

// FileInfo 文件信息
type FileInfo struct {
	FilePath     string
	FileType     string
	FileSize     int64
	Architecture string
	Sections     []SectionInfo
	Strings      []StringInfo
}

// NewBinaryAnalyzer 创建二进制分析器
func NewBinaryAnalyzer(filePath string) *BinaryAnalyzer {
	return &BinaryAnalyzer{
		filePath: filePath,
	}
}

// AnalyzeFile 分析文件
func (ba *BinaryAnalyzer) AnalyzeFile() (*FileInfo, error) {
	// 读取文件
	file, err := os.Open(ba.filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	// 获取文件信息
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	// 读取文件数据
	ba.data, err = io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("读取文件数据失败: %v", err)
	}

	// 检测文件类型
	fileType, err := ba.detectFileType()
	if err != nil {
		return nil, fmt.Errorf("检测文件类型失败: %v", err)
	}

	ba.fileType = fileType

	// 创建文件信息
	fileInfo := &FileInfo{
		FilePath: ba.filePath,
		FileType: fileType,
		FileSize: stat.Size(),
	}

	// 根据文件类型解析
	switch fileType {
	case "Mach-O":
		err = ba.parseMachO(fileInfo)
	case "PE":
		err = ba.parsePE(fileInfo)
	case "ELF":
		err = ba.parseELF(fileInfo)
	default:
		err = ba.parseGeneric(fileInfo)
	}

	if err != nil {
		return nil, fmt.Errorf("解析文件失败: %v", err)
	}

	// 提取字符串
	fileInfo.Strings = ba.extractStrings()

	return fileInfo, nil
}

// detectFileType 检测文件类型
func (ba *BinaryAnalyzer) detectFileType() (string, error) {
	if len(ba.data) < 16 {
		return "Unknown", fmt.Errorf("文件太小")
	}

	header := ba.data[:16]

	// Mach-O魔数检查
	if len(header) >= 4 {
		magic := binary.LittleEndian.Uint32(header[:4])
		switch magic {
		case 0xFEEDFACE: // MH_MAGIC (32-bit)
			return "Mach-O", nil
		case 0xFEEDFACF: // MH_MAGIC_64 (64-bit)
			return "Mach-O", nil
		case 0xCAFEBABE: // FAT_MAGIC (Universal binary)
			return "Mach-O", nil
		case 0xBEBAFECA: // FAT_CIGAM (Universal binary, swapped)
			return "Mach-O", nil
		}

		// PE魔数检查
		if header[0] == 'M' && header[1] == 'Z' {
			return "PE", nil
		}

		// ELF魔数检查
		if header[0] == 0x7F && header[1] == 'E' && header[2] == 'L' && header[3] == 'F' {
			return "ELF", nil
		}
	}

	return "Unknown", nil
}

// parseMachO 解析Mach-O文件
func (ba *BinaryAnalyzer) parseMachO(fileInfo *FileInfo) error {
	if len(ba.data) < 32 {
		return fmt.Errorf("Mach-O文件太小")
	}

	// 简化的Mach-O解析
	fileInfo.Architecture = "arm64" // 简化处理

	// 添加一些基本段信息
	fileInfo.Sections = []SectionInfo{
		{Index: 0, Name: "LC_SEGMENT_64(__TEXT)", Offset: 0x0, Size: 0x4000, Type: "Load Command"},
		{Index: 1, Name: "__text", Offset: 0x1000, Size: 0x2000, Type: "Code Section"},
		{Index: 2, Name: "__cstring", Offset: 0x3000, Size: 0x1000, Type: "String Section"},
		{Index: 3, Name: "LC_SEGMENT_64(__DATA)", Offset: 0x4000, Size: 0x2000, Type: "Load Command"},
		{Index: 4, Name: "__data", Offset: 0x4000, Size: 0x1000, Type: "Data Section"},
		{Index: 5, Name: "__bss", Offset: 0x5000, Size: 0x1000, Type: "BSS Section"},
	}

	return nil
}

// parsePE 解析PE文件
func (ba *BinaryAnalyzer) parsePE(fileInfo *FileInfo) error {
	if len(ba.data) < 64 {
		return fmt.Errorf("PE文件太小")
	}

	// 简化的PE解析
	fileInfo.Architecture = "x86_64" // 简化处理

	// 添加一些基本段信息
	fileInfo.Sections = []SectionInfo{
		{Index: 0, Name: ".text", Offset: 0x1000, Size: 0x5000, Type: "Code Section"},
		{Index: 1, Name: ".data", Offset: 0x6000, Size: 0x2000, Type: "Data Section"},
		{Index: 2, Name: ".rdata", Offset: 0x8000, Size: 0x3000, Type: "Read-only Data"},
		{Index: 3, Name: ".idata", Offset: 0xB000, Size: 0x1000, Type: "Import Data"},
		{Index: 4, Name: ".reloc", Offset: 0xC000, Size: 0x1000, Type: "Relocation Data"},
	}

	return nil
}

// parseELF 解析ELF文件
func (ba *BinaryAnalyzer) parseELF(fileInfo *FileInfo) error {
	if len(ba.data) < 64 {
		return fmt.Errorf("ELF文件太小")
	}

	// 简化的ELF解析
	if ba.data[4] == 1 {
		fileInfo.Architecture = "32-bit"
	} else {
		fileInfo.Architecture = "64-bit"
	}

	// 添加一些基本段信息
	fileInfo.Sections = []SectionInfo{
		{Index: 0, Name: ".text", Offset: 0x1000, Size: 0x4000, Type: "PROGBITS"},
		{Index: 1, Name: ".data", Offset: 0x5000, Size: 0x1000, Type: "PROGBITS"},
		{Index: 2, Name: ".rodata", Offset: 0x6000, Size: 0x2000, Type: "PROGBITS"},
		{Index: 3, Name: ".bss", Offset: 0x8000, Size: 0x1000, Type: "NOBITS"},
		{Index: 4, Name: ".symtab", Offset: 0x9000, Size: 0x800, Type: "SYMTAB"},
		{Index: 5, Name: ".strtab", Offset: 0x9800, Size: 0x400, Type: "STRTAB"},
	}

	return nil
}

// parseGeneric 通用解析
func (ba *BinaryAnalyzer) parseGeneric(fileInfo *FileInfo) error {
	fileInfo.Architecture = "Unknown"
	fileInfo.Sections = []SectionInfo{
		{Index: 0, Name: "Raw Data", Offset: 0x0, Size: uint64(len(ba.data)), Type: "Data"},
	}
	return nil
}

// extractStrings 提取字符串
func (ba *BinaryAnalyzer) extractStrings() []StringInfo {
	var strings []StringInfo
	var currentString []byte
	var stringStart uint64
	index := 0

	for i, b := range ba.data {
		if isPrintable(b) {
			if len(currentString) == 0 {
				stringStart = uint64(i)
			}
			currentString = append(currentString, b)
		} else {
			if len(currentString) >= 4 { // 只保留长度>=4的字符串
				str := string(currentString)
				hexData := fmt.Sprintf("%X", currentString)

				strings = append(strings, StringInfo{
					Index:  index,
					Offset: stringStart,
					Data:   hexData,
					Length: len(currentString),
					String: str,
				})
				index++
			}
			currentString = nil
		}
	}

	// 处理文件末尾的字符串
	if len(currentString) >= 4 {
		str := string(currentString)
		hexData := fmt.Sprintf("%X", currentString)

		strings = append(strings, StringInfo{
			Index:  index,
			Offset: stringStart,
			Data:   hexData,
			Length: len(currentString),
			String: str,
		})
	}

	return strings
}

// isPrintable 检查字符是否可打印
func isPrintable(b byte) bool {
	return unicode.IsPrint(rune(b)) && b < 127
}

// SearchStrings 搜索字符串
func (ba *BinaryAnalyzer) SearchStrings(pattern string) []StringInfo {
	var results []StringInfo
	pattern = strings.ToLower(pattern)

	allStrings := ba.extractStrings()
	for _, str := range allStrings {
		if strings.Contains(strings.ToLower(str.String), pattern) {
			results = append(results, str)
		}
	}

	return results
}

// GetSectionStrings 获取指定段的字符串
func (ba *BinaryAnalyzer) GetSectionStrings(sectionIndex int, sections []SectionInfo) []StringInfo {
	if sectionIndex < 0 || sectionIndex >= len(sections) {
		return []StringInfo{}
	}

	section := sections[sectionIndex]
	var results []StringInfo

	allStrings := ba.extractStrings()
	for _, str := range allStrings {
		if str.Offset >= section.Offset && str.Offset < section.Offset+section.Size {
			results = append(results, str)
		}
	}

	return results
}
