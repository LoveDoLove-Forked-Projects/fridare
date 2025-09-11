package core

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"fmt"
	"os"
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
	Index      int
	Name       string
	Offset     uint64
	Size       uint64
	Type       string
	ArchIndex  int    // 架构索引（Fat Mach-O使用）
	ArchOffset uint64 // 架构在文件中的偏移（Fat Mach-O使用）
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
	FilePath        string
	FileType        string
	FileSize        int64
	Architecture    string
	DetailedInfo    string // 详细的架构信息，来自arch_desc.go
	IsFatMachO      bool   // 是否为 Fat Mach-O 文件
	Sections        []SectionInfo
	SelectedSection *SectionInfo // 当前选中的段
	SectionData     []byte       // 选中段的数据
}

// NewBinaryAnalyzer 创建二进制分析器
func NewBinaryAnalyzer(filePath string) *BinaryAnalyzer {
	return &BinaryAnalyzer{
		filePath: filePath,
	}
}

// AnalyzeFile 分析文件
func (ba *BinaryAnalyzer) AnalyzeFile() (*FileInfo, error) {
	// 获取文件信息
	file, err := os.Open(ba.filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	// 使用hexreplace.go中的检测和打开逻辑
	binFile, format, err := detectAndOpenFile(ba.filePath)
	if err != nil {
		return nil, fmt.Errorf("检测文件格式失败: %v", err)
	}

	// 创建文件信息
	fileInfo := &FileInfo{
		FilePath: ba.filePath,
		FileType: formatToString(format),
		FileSize: stat.Size(),
	}

	// 获取详细的架构信息（使用arch_desc.go）
	fileInfo.DetailedInfo = describeArch(binFile, format)

	// 根据文件类型解析段信息
	switch f := binFile.(type) {
	case *macho.File:
		fileInfo.Architecture = cpuTypeToString(f.Cpu)
		fileInfo.IsFatMachO = false
		fileInfo.Sections = ba.parseMachOSections(f)
	case *macho.FatFile:
		fileInfo.Architecture = "Universal Binary (Multiple Architectures)"
		fileInfo.IsFatMachO = true
		fileInfo.Sections = ba.parseFatMachOSections(f)
	case *elf.File:
		fileInfo.Architecture = f.Machine.String()
		fileInfo.IsFatMachO = false
		fileInfo.Sections = ba.parseELFSections(f)
	case *pe.File:
		fileInfo.Architecture = fmt.Sprintf("Machine: %d", f.Machine)
		fileInfo.IsFatMachO = false
		fileInfo.Sections = ba.parsePESections(f)
	default:
		return nil, fmt.Errorf("不支持的文件格式")
	}

	return fileInfo, nil
}

// parseMachOSections 解析Mach-O单架构文件的段信息
func (ba *BinaryAnalyzer) parseMachOSections(f *macho.File) []SectionInfo {
	var sections []SectionInfo
	index := 0

	// 遍历所有段
	for _, sect := range f.Sections {
		sections = append(sections, SectionInfo{
			Index:  index,
			Name:   sect.Name,
			Offset: uint64(sect.Offset),
			Size:   sect.Size,
			Type:   fmt.Sprintf("Flags: 0x%X", sect.Flags),
		})
		index++
	}

	return sections
}

// parseFatMachOSections 解析Fat Mach-O文件的段信息
func (ba *BinaryAnalyzer) parseFatMachOSections(f *macho.FatFile) []SectionInfo {
	var sections []SectionInfo
	index := 0

	// 遍历所有架构
	for archIndex, arch := range f.Arches {
		// 添加架构信息作为一个节点
		sections = append(sections, SectionInfo{
			Index:      index,
			Name:       fmt.Sprintf("Architecture %d: %s", archIndex, arch.Cpu.String()),
			Offset:     uint64(arch.Offset),
			Size:       uint64(arch.Size),
			Type:       "Architecture",
			ArchIndex:  archIndex,
			ArchOffset: uint64(arch.Offset),
		})
		index++

		// 添加该架构的段信息
		for _, sect := range arch.Sections {
			sections = append(sections, SectionInfo{
				Index:      index,
				Name:       sect.Name, // 移除前缀空格
				Offset:     uint64(arch.Offset + sect.Offset),
				Size:       sect.Size,
				Type:       fmt.Sprintf("Flags: 0x%X", sect.Flags),
				ArchIndex:  archIndex,
				ArchOffset: uint64(arch.Offset),
			})
			index++
		}
	}

	return sections
}

// parseELFSections 解析ELF文件的段信息
func (ba *BinaryAnalyzer) parseELFSections(f *elf.File) []SectionInfo {
	var sections []SectionInfo

	// 遍历所有段
	for i, sect := range f.Sections {
		sections = append(sections, SectionInfo{
			Index:  i,
			Name:   sect.Name,
			Offset: sect.Offset,
			Size:   sect.Size,
			Type:   sect.Type.String(),
		})
	}

	return sections
}

// parsePESections 解析PE文件的段信息
func (ba *BinaryAnalyzer) parsePESections(f *pe.File) []SectionInfo {
	var sections []SectionInfo

	// 遍历所有段
	for i, sect := range f.Sections {
		sections = append(sections, SectionInfo{
			Index:  i,
			Name:   sect.Name,
			Offset: uint64(sect.Offset),
			Size:   uint64(sect.Size),
			Type:   fmt.Sprintf("VAddr: 0x%X", sect.VirtualAddress),
		})
	}

	return sections
}

// GetSectionData 获取指定段的数据
func (ba *BinaryAnalyzer) GetSectionData(filePath string, sectionIndex int, sections []SectionInfo) ([]byte, error) {
	if sectionIndex < 0 || sectionIndex >= len(sections) {
		return nil, fmt.Errorf("无效的段索引")
	}

	// 使用hexreplace.go中的检测和打开逻辑
	binFile, format, err := detectAndOpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}

	// 根据文件类型获取段数据
	switch f := binFile.(type) {
	case *macho.File:
		if sectionIndex < len(f.Sections) {
			return f.Sections[sectionIndex].Data()
		}
	case *macho.FatFile:
		// Fat文件需要特殊处理，根据段索引找到对应的架构和段
		if sectionIndex < len(sections) {
			section := sections[sectionIndex]

			// 如果是架构节点，返回空数据
			if section.Type == "Architecture" {
				return []byte{}, nil
			}

			// 找到对应的架构
			if section.ArchIndex < len(f.Arches) {
				arch := f.Arches[section.ArchIndex]

				// 在该架构中找到对应的段
				for _, sect := range arch.Sections {
					if sect.Name == section.Name {
						return sect.Data()
					}
				}
			}
		}
		return nil, fmt.Errorf("未找到 Fat Mach-O 段数据")
	case *elf.File:
		if sectionIndex < len(f.Sections) {
			return f.Sections[sectionIndex].Data()
		}
	case *pe.File:
		if sectionIndex < len(f.Sections) {
			return f.Sections[sectionIndex].Data()
		}
	default:
		return nil, fmt.Errorf("不支持的文件格式: %v", format)
	}

	return nil, fmt.Errorf("未找到段数据")
}

// isPrintable 检查字符是否可打印
func isPrintable(b byte) bool {
	return unicode.IsPrint(rune(b)) && b < 127
}

// ExtractStringsFromData 从数据中提取字符串
func (ba *BinaryAnalyzer) ExtractStringsFromData(data []byte, baseOffset uint64) []StringInfo {
	var strings []StringInfo
	var currentString []byte
	var stringStart uint64
	index := 0

	for i, b := range data {
		if isPrintable(b) {
			if len(currentString) == 0 {
				stringStart = baseOffset + uint64(i)
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

	// 处理末尾的字符串
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
