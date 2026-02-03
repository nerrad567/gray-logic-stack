package etsimport

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Parser configuration constants.
const (
	// MaxFileSize is the maximum allowed file size (50MB).
	MaxFileSize = 50 * 1024 * 1024

	// MaxParseTime is the maximum time allowed for parsing.
	MaxParseTime = 60 * time.Second

	// importIDBytes is the number of random bytes for import IDs.
	importIDBytes = 8

	// Format constants.
	formatKNXProj = "knxproj"
	formatXML     = "xml"
	formatCSV     = "csv"

	// Common strings.
	typeSwitch = "switch"
	typeSensor = "sensor"

	// Regex match counts.
	regexMatchCount3 = 3
	regexMatchCount2 = 2

	// Maximum confidence cap.
	maxConfidence = 0.99
)

// Parser parses ETS project files and detects device configurations.
type Parser struct {
	// detectionRules are the device detection rules.
	detectionRules []DetectionRule
}

// NewParser creates a new ETS parser with default detection rules.
func NewParser() *Parser {
	return &Parser{
		detectionRules: DefaultDetectionRules(),
	}
}

// ParseBytes parses an ETS project from a byte slice.
func (p *Parser) ParseBytes(data []byte, filename string) (*ParseResult, error) {
	if len(data) > MaxFileSize {
		return nil, ErrFileTooLarge
	}

	result := &ParseResult{
		ImportID:   generateImportID(),
		SourceFile: filepath.Base(filename),
		ParsedAt:   time.Now().UTC(),
	}

	// Detect format and parse
	ext := strings.ToLower(filepath.Ext(filename))
	// Debug logging - will be visible in server logs
	fmt.Printf("[DEBUG] ParseBytes: filename=%q, ext=%q, dataLen=%d\n", filename, ext, len(data))

	// projectXMLData stores the raw 0.xml bytes for Tier 1 extraction
	var projectXMLData []byte

	switch ext {
	case ".knxproj":
		xmlData, err := p.parseKNXProjWithXML(data, result)
		if err != nil {
			return nil, err
		}
		projectXMLData = xmlData
		result.Format = formatKNXProj

	case ".xml":
		if err := p.parseXML(data, result); err != nil {
			return nil, err
		}
		projectXMLData = data
		result.Format = formatXML

	case ".csv":
		if err := p.parseCSV(data, result); err != nil {
			return nil, err
		}
		result.Format = formatCSV

	default:
		// Try to detect format from content
		if isZipFile(data) {
			xmlData, err := p.parseKNXProjWithXML(data, result)
			if err != nil {
				return nil, err
			}
			projectXMLData = xmlData
			result.Format = formatKNXProj
		} else if isXMLFile(data) {
			if err := p.parseXML(data, result); err != nil {
				return nil, err
			}
			projectXMLData = data
			result.Format = formatXML
		} else {
			return nil, ErrInvalidFile
		}
	}

	// Tier 1: Function-based classification (requires project XML with
	// Topology, ManufacturerData, and Trades sections)
	var consumedGAIDs map[string]bool
	if projectXMLData != nil && len(result.UnmappedAddresses) > 0 {
		consumedGAIDs = p.extractFunctionDevices(projectXMLData, result)
		if len(consumedGAIDs) > 0 {
			// Remove consumed GAs so Tier 2 doesn't re-detect them.
			// Build GA ID→Address map by re-parsing GA IDs.
			gaIDToAddr := p.buildGAIDToAddrMap(projectXMLData)
			result.UnmappedAddresses = removeConsumedGAs(
				result.UnmappedAddresses, consumedGAIDs, gaIDToAddr)
		}
	}

	// Tier 2: DPT-based device detection on remaining unmapped GAs
	p.detectDevices(result)

	// Extract building locations from hierarchy paths
	p.extractLocations(result)

	// Calculate statistics
	p.calculateStatistics(result)

	return result, nil
}

// parseKNXProj extracts and parses a .knxproj ZIP archive.
func (p *Parser) parseKNXProj(data []byte, result *ParseResult) error {
	_, err := p.parseKNXProjWithXML(data, result)
	return err
}

// parseKNXProjWithXML extracts and parses a .knxproj ZIP archive, returning
// the raw project XML (0.xml) for use by Tier 1 function extraction.
func (p *Parser) parseKNXProjWithXML(data []byte, result *ParseResult) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCorruptArchive, err)
	}

	var groupAddressesXML []byte
	var projectXML []byte

	// Debug: list all files in the archive
	fmt.Printf("[DEBUG] parseKNXProj: archive contains %d files\n", len(reader.File))
	for i, f := range reader.File {
		if i < 20 { // Only print first 20 to avoid spam
			fmt.Printf("[DEBUG]   file[%d]: %s\n", i, f.Name)
		}
	}

	// Find relevant XML files in the archive
	for _, file := range reader.File {
		name := strings.ToLower(filepath.Base(file.Name))

		switch {
		case name == "groupaddresses.xml" || strings.HasSuffix(strings.ToLower(file.Name), "/groupaddresses.xml"):
			content, err := readZipFile(file)
			if err != nil {
				return nil, fmt.Errorf("reading GroupAddresses.xml: %w", err)
			}
			groupAddressesXML = content

		case name == "0.xml" && projectXML == nil:
			content, err := readZipFile(file)
			if err != nil {
				return nil, fmt.Errorf("reading project XML: %w", err)
			}
			projectXML = content

		case name == "project.xml":
			content, err := readZipFile(file)
			if err != nil {
				return nil, fmt.Errorf("reading project.xml: %w", err)
			}
			// Try to extract ETS version
			result.ETSVersion = extractETSVersion(content)
		}
	}

	if groupAddressesXML == nil {
		// Try parsing project XML for group addresses
		if projectXML != nil {
			fmt.Printf("[DEBUG] No GroupAddresses.xml found, trying to parse 0.xml (%d bytes)\n", len(projectXML))
			err := p.parseProjectXML(projectXML, result)
			if err != nil {
				fmt.Printf("[DEBUG] parseProjectXML failed: %v\n", err)
				return nil, err
			}
			return projectXML, nil
		}
		fmt.Printf("[DEBUG] No GroupAddresses.xml and no 0.xml found\n")
		return nil, ErrNoGroupAddresses
	}

	if err := p.parseGroupAddressesXML(groupAddressesXML, result); err != nil {
		return nil, err
	}
	return projectXML, nil
}

// parseGroupAddressesXML parses the GroupAddresses.xml file from ETS.
func (p *Parser) parseGroupAddressesXML(data []byte, result *ParseResult) error {
	// ETS GroupAddresses.xml structure
	type xmlLink struct {
		RefID string `xml:"RefId,attr"`
	}
	type xmlLinks struct {
		Links []xmlLink `xml:"Link"`
	}
	type xmlGroupAddress struct {
		ID          string   `xml:"Id,attr"`
		Address     string   `xml:"Address,attr"`
		Name        string   `xml:"Name,attr"`
		DPT         string   `xml:"DatapointType,attr"`
		Description string   `xml:"Description,attr"`
		Links       xmlLinks `xml:"Links"`
	}
	type xmlGroupRange struct {
		Name      string            `xml:"Name,attr"`
		Address   string            `xml:"Address,attr"`
		Ranges    []xmlGroupRange   `xml:"GroupRange"`
		Addresses []xmlGroupAddress `xml:"GroupAddress"`
	}
	type xmlGroupAddresses struct {
		XMLName xml.Name        `xml:"GroupAddresses"`
		Ranges  []xmlGroupRange `xml:"GroupRange"`
	}

	var doc xmlGroupAddresses
	if err := xml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidFile, err)
	}

	// Recursively extract all group addresses
	var extractAddresses func(ranges []xmlGroupRange, path string)
	extractAddresses = func(ranges []xmlGroupRange, path string) {
		for _, r := range ranges {
			currentPath := r.Name
			if path != "" {
				currentPath = path + " > " + r.Name
			}

			// Extract addresses at this level
			for _, addr := range r.Addresses {
				ga := GroupAddress{
					Address:     normaliseGA(addr.Address),
					Name:        addr.Name,
					DPT:         normaliseDPT(addr.DPT),
					Description: addr.Description,
					Location:    currentPath,
				}

				// Extract linked device references
				for _, link := range addr.Links.Links {
					ga.LinkedDevices = append(ga.LinkedDevices, link.RefID)
				}

				result.UnmappedAddresses = append(result.UnmappedAddresses, ga)
			}

			// Recurse into sub-ranges
			extractAddresses(r.Ranges, currentPath)
		}
	}

	extractAddresses(doc.Ranges, "")

	if len(result.UnmappedAddresses) == 0 {
		return ErrNoGroupAddresses
	}

	return nil
}

// parseProjectXML parses a 0.xml project file for group addresses.
func (p *Parser) parseProjectXML(data []byte, result *ParseResult) error {
	// ETS 0.xml has group addresses nested deeply. Try multiple paths.

	// Structure 1: Standard ETS5/6 format
	type xmlGA struct {
		Address string `xml:"Address,attr"`
		Name    string `xml:"Name,attr"`
		DPT     string `xml:"DatapointType,attr"`
	}
	type xmlGroupRange struct {
		Name      string          `xml:"Name,attr"`
		Ranges    []xmlGroupRange `xml:"GroupRange"`
		Addresses []xmlGA         `xml:"GroupAddress"`
	}
	type xmlProject struct {
		XMLName xml.Name        `xml:"KNX"`
		Ranges  []xmlGroupRange `xml:"Project>Installations>Installation>GroupAddresses>GroupRanges>GroupRange"`
	}

	var doc xmlProject
	if err := xml.Unmarshal(data, &doc); err != nil {
		fmt.Printf("[DEBUG] parseProjectXML: XML unmarshal failed: %v\n", err)
		// Try alternative structure
		return p.parseGenericXML(data, result)
	}

	fmt.Printf("[DEBUG] parseProjectXML: found %d top-level GroupRanges\n", len(doc.Ranges))

	// Recursively extract addresses from nested GroupRanges
	var extractFromRanges func(ranges []xmlGroupRange, path string)
	extractFromRanges = func(ranges []xmlGroupRange, path string) {
		for _, r := range ranges {
			currentPath := r.Name
			if path != "" {
				currentPath = path + " > " + r.Name
			}
			fmt.Printf("[DEBUG]   GroupRange: %s, addresses: %d, sub-ranges: %d\n",
				currentPath, len(r.Addresses), len(r.Ranges))

			for _, addr := range r.Addresses {
				result.UnmappedAddresses = append(result.UnmappedAddresses, GroupAddress{
					Address:  normaliseGA(addr.Address),
					Name:     addr.Name,
					DPT:      normaliseDPT(addr.DPT),
					Location: currentPath,
				})
			}
			extractFromRanges(r.Ranges, currentPath)
		}
	}

	extractFromRanges(doc.Ranges, "")

	fmt.Printf("[DEBUG] parseProjectXML: extracted %d addresses\n", len(result.UnmappedAddresses))

	if len(result.UnmappedAddresses) == 0 {
		fmt.Printf("[DEBUG] parseProjectXML: no addresses found, trying generic XML parser\n")
		return p.parseGenericXML(data, result)
	}

	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Tier 1: Function-based device classification from ETS metadata
// ─────────────────────────────────────────────────────────────────────────────

// XML structures for ETS Topology, ManufacturerData, and Functions.

type xmlDeviceInstance struct {
	ID                    string            `xml:"Id,attr"`
	Name                  string            `xml:"Name,attr"`
	IndividualAddress     string            `xml:"IndividualAddress,attr"`
	ProductRefId          string            `xml:"ProductRefId,attr"`
	ApplicationProgramRef string            `xml:"ApplicationProgramRef,attr"`
	ComObjectRefs         []xmlComObjectRef `xml:"ComObjectInstanceRefs>ComObjectInstanceRef"`
}

type xmlComObjectRef struct {
	RefID string         `xml:"RefId,attr"`
	DPT   string         `xml:"DatapointType,attr"`
	Send  []xmlConnector `xml:"Connectors>Send"`
}

type xmlConnector struct {
	GARefID string `xml:"GroupAddressRefId,attr"`
}

type xmlFunction struct {
	ID      string      `xml:"Id,attr"`
	Name    string      `xml:"Name,attr"`
	Type    string      `xml:"Type,attr"`
	Comment string      `xml:"Comment,attr"`
	GARefs  []xmlGARef  `xml:"GroupAddressRefs>GroupAddressRef"`
	LocRef  []xmlLocRef `xml:"LocationReference"`
}

type xmlGARef struct {
	RefID string `xml:"RefId,attr"`
	Name  string `xml:"Name,attr"`
}

type xmlLocRef struct {
	RefID string `xml:"RefId,attr"`
}

type xmlManufacturer struct {
	ID          string          `xml:"Id,attr"`
	Name        string          `xml:"Name,attr"`
	AppPrograms []xmlAppProgram `xml:"ApplicationProgram"`
	Hardware    []xmlHardware   `xml:"Hardware"`
}

type xmlAppProgram struct {
	ID   string `xml:"Id,attr"`
	Name string `xml:"Name,attr"`
}

type xmlHardware struct {
	ID   string `xml:"Id,attr"`
	Name string `xml:"Name,attr"`
}

type xmlTrade struct {
	Functions []xmlFunction `xml:"Function"`
}

type xmlBuildingLocation struct {
	ID       string                `xml:"Id,attr"`
	Name     string                `xml:"Name,attr"`
	Type     string                `xml:"Type,attr"`
	Children []xmlBuildingLocation `xml:"Location"`
}

// xmlProjectFull extends the basic project parse to include Topology,
// ManufacturerData, Trades (Functions), and Building Locations.
type xmlProjectFull struct {
	XMLName       xml.Name              `xml:"KNX"`
	Manufacturers []xmlManufacturer     `xml:"ManufacturerData>Manufacturer"`
	Devices       []xmlDeviceInstance   `xml:"Project>Installations>Installation>Topology>Area>Line>DeviceInstance"`
	Trades        []xmlTrade            `xml:"Project>Installations>Installation>Trades>Trade"`
	Locations     []xmlBuildingLocation `xml:"Project>Installations>Installation>Locations>Location"`
}

// extractFunctionDevices performs Tier 1 classification: it parses Functions
// (from the Trades section), Topology (DeviceInstances), and ManufacturerData,
// then maps ETS Function Types directly to GLCore device types.
//
// It returns the set of GA IDs that were consumed by Tier 1 devices, so
// Tier 2 (DPT-based detection) can skip them.
func (p *Parser) extractFunctionDevices(data []byte, result *ParseResult) map[string]bool {
	var doc xmlProjectFull
	if err := xml.Unmarshal(data, &doc); err != nil {
		fmt.Printf("[DEBUG] extractFunctionDevices: unmarshal failed: %v\n", err)
		return nil
	}

	if len(doc.Trades) == 0 {
		fmt.Printf("[DEBUG] extractFunctionDevices: no Trades/Functions found\n")
		return nil
	}

	// Build lookup maps
	// GA ID → GroupAddress (from already-parsed unmapped addresses)
	gaByID := make(map[string]*GroupAddress)
	gaByAddr := make(map[string]*GroupAddress)
	for i := range result.UnmappedAddresses {
		ga := &result.UnmappedAddresses[i]
		gaByAddr[ga.Address] = ga
	}

	// We need the GA XML IDs. Re-parse just the GA section to get ID→Address mapping.
	type xmlGAEntry struct {
		ID      string `xml:"Id,attr"`
		Address string `xml:"Address,attr"`
		Name    string `xml:"Name,attr"`
		DPT     string `xml:"DatapointType,attr"`
	}
	type xmlGARangeEntry struct {
		Addresses []xmlGAEntry      `xml:"GroupAddress"`
		Ranges    []xmlGARangeEntry `xml:"GroupRange"`
	}
	type xmlGAProject struct {
		XMLName xml.Name          `xml:"KNX"`
		Ranges  []xmlGARangeEntry `xml:"Project>Installations>Installation>GroupAddresses>GroupRanges>GroupRange"`
	}

	var gaDoc xmlGAProject
	if err := xml.Unmarshal(data, &gaDoc); err == nil {
		var walkRanges func(ranges []xmlGARangeEntry)
		walkRanges = func(ranges []xmlGARangeEntry) {
			for _, r := range ranges {
				for _, ga := range r.Addresses {
					addr := normaliseGA(ga.Address)
					if existing, ok := gaByAddr[addr]; ok {
						gaByID[ga.ID] = existing
					}
				}
				walkRanges(r.Ranges)
			}
		}
		walkRanges(gaDoc.Ranges)
	}

	// Manufacturer ID → name
	mfrNames := make(map[string]string)
	for _, mfr := range doc.Manufacturers {
		mfrNames[mfr.ID] = mfr.Name
	}

	// App program ID → name
	appNames := make(map[string]string)
	for _, mfr := range doc.Manufacturers {
		for _, app := range mfr.AppPrograms {
			appNames[app.ID] = app.Name
		}
	}

	// Hardware ID → model name
	hwNames := make(map[string]string)
	for _, mfr := range doc.Manufacturers {
		for _, hw := range mfr.Hardware {
			hwNames[hw.ID] = hw.Name
		}
	}

	// Device ID → DeviceInstance (for linking topology metadata to functions)
	deviceByID := make(map[string]*xmlDeviceInstance)
	for i := range doc.Devices {
		deviceByID[doc.Devices[i].ID] = &doc.Devices[i]
	}

	// Build GA→DeviceInstance map via ComObjectRefs
	gaRefToDevice := make(map[string]*xmlDeviceInstance)
	for i := range doc.Devices {
		dev := &doc.Devices[i]
		for _, co := range dev.ComObjectRefs {
			for _, send := range co.Send {
				gaRefToDevice[send.GARefID] = dev
			}
		}
	}

	// Building Location ID → path (e.g. "L-FD02E366" → "Ground Floor > Distribution Board")
	// Used to resolve Function LocationReferences to room/area paths.
	locIDToPath := make(map[string]string)
	var walkLocations func(locs []xmlBuildingLocation, parentPath string)
	walkLocations = func(locs []xmlBuildingLocation, parentPath string) {
		for _, loc := range locs {
			path := loc.Name
			if parentPath != "" {
				path = parentPath + " > " + loc.Name
			}
			locIDToPath[loc.ID] = path
			walkLocations(loc.Children, path)
		}
	}
	walkLocations(doc.Locations, "")

	// Process Functions from Trades
	consumedGAIDs := make(map[string]bool)
	tier1Count := 0

	for _, trade := range doc.Trades {
		for _, fn := range trade.Functions {
			deviceType, domain, confidence := functionTypeToDeviceType(fn.Type, fn.Comment)
			if deviceType == "" {
				continue // Unknown function type — skip
			}

			// Resolve GroupAddressRefs → actual GA data
			var addresses []DeviceAddress
			var sourceLocation string
			for _, gaRef := range fn.GARefs {
				ga, ok := gaByID[gaRef.RefID]
				if !ok {
					continue
				}
				funcName := gaRef.Name
				if funcName == "" {
					funcName = inferFunctionFromName(ga.Name)
					if funcName == "" {
						funcName = inferFunctionFromDPTValue(ga.DPT)
					}
				}
				addresses = append(addresses, DeviceAddress{
					GA:                ga.Address,
					Name:              ga.Name,
					DPT:               ga.DPT,
					SuggestedFunction: funcName,
					SuggestedFlags:    inferFlags(ga.DPT, ga.Name),
					Description:       ga.Description,
				})
				if sourceLocation == "" {
					sourceLocation = ga.Location
				}
				consumedGAIDs[gaRef.RefID] = true
			}

			if len(addresses) == 0 {
				continue // No resolvable GAs — skip
			}

			// Prefer the Function's own LocationReference over the GA path.
			// GAs belong to the room where the load is (e.g. Living Room),
			// but the Function's LocationRef points to where the device
			// physically lives (e.g. Distribution Board).
			if len(fn.LocRef) > 0 {
				if path, ok := locIDToPath[fn.LocRef[0].RefID]; ok {
					sourceLocation = path
				}
			}

			// Try to find linked DeviceInstance for metadata
			var manufacturer, productModel, appProgram, indAddr string
			for _, gaRef := range fn.GARefs {
				if dev, ok := gaRefToDevice[gaRef.RefID]; ok {
					indAddr = dev.IndividualAddress
					// Extract manufacturer from ProductRefId prefix (e.g., "M-0083_H-0001-HP-0001")
					if parts := strings.SplitN(dev.ProductRefId, "_", 2); len(parts) > 0 {
						manufacturer = mfrNames[parts[0]]
					}
					// Extract hardware model from ProductRefId
					hwID := ""
					if idx := strings.Index(dev.ProductRefId, "-HP-"); idx > 0 {
						hwID = dev.ProductRefId[:idx]
					}
					if hwID != "" {
						productModel = hwNames[hwID]
					}
					appProgram = appNames[dev.ApplicationProgramRef]
					break
				}
			}

			device := DetectedDevice{
				SuggestedID:        generateSlug(fn.Name),
				SuggestedName:      cleanName(fn.Name),
				DetectedType:       deviceType,
				Confidence:         confidence,
				SuggestedDomain:    domain,
				Addresses:          addresses,
				SourceLocation:     sourceLocation,
				Manufacturer:       manufacturer,
				ProductModel:       productModel,
				ApplicationProgram: appProgram,
				IndividualAddress:  indAddr,
				FunctionType:       fn.Type,
				FunctionComment:    fn.Comment,
			}

			if sourceLocation != "" {
				device.SuggestedRoom = extractRoomFromLocation(sourceLocation)
				device.SuggestedArea = extractAreaFromLocation(sourceLocation)
			}

			result.Devices = append(result.Devices, device)
			tier1Count++
		}
	}

	fmt.Printf("[DEBUG] extractFunctionDevices: Tier 1 classified %d devices, consumed %d GA refs\n",
		tier1Count, len(consumedGAIDs))

	if len(consumedGAIDs) == 0 {
		return nil
	}
	return consumedGAIDs
}

// removeConsumedGAs filters out group addresses that were already consumed
// by Tier 1 (function-based) classification.
func removeConsumedGAs(addresses []GroupAddress, consumedIDs map[string]bool, gaIDToAddr map[string]string) []GroupAddress {
	if len(consumedIDs) == 0 {
		return addresses
	}

	// Build set of consumed GA addresses
	consumedAddrs := make(map[string]bool)
	for gaID := range consumedIDs {
		if addr, ok := gaIDToAddr[gaID]; ok {
			consumedAddrs[addr] = true
		}
	}

	var remaining []GroupAddress
	for _, ga := range addresses {
		if !consumedAddrs[ga.Address] {
			remaining = append(remaining, ga)
		}
	}
	return remaining
}

// buildGAIDToAddrMap parses the project XML to build a map from GA XML IDs
// to normalised GA address strings. Used to correlate consumed GA IDs from
// Tier 1 with actual addresses in the unmapped pool.
func (p *Parser) buildGAIDToAddrMap(data []byte) map[string]string {
	type xmlGAE struct {
		ID      string `xml:"Id,attr"`
		Address string `xml:"Address,attr"`
	}
	type xmlGRE struct {
		Addresses []xmlGAE `xml:"GroupAddress"`
		Ranges    []xmlGRE `xml:"GroupRange"`
	}
	type xmlDoc struct {
		XMLName xml.Name `xml:"KNX"`
		Ranges  []xmlGRE `xml:"Project>Installations>Installation>GroupAddresses>GroupRanges>GroupRange"`
	}

	var doc xmlDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil
	}

	result := make(map[string]string)
	var walk func(ranges []xmlGRE)
	walk = func(ranges []xmlGRE) {
		for _, r := range ranges {
			for _, ga := range r.Addresses {
				result[ga.ID] = normaliseGA(ga.Address)
			}
			walk(r.Ranges)
		}
	}
	walk(doc.Ranges)
	return result
}

// parseGenericXML attempts to parse any XML with group address-like content.
func (p *Parser) parseGenericXML(data []byte, result *ParseResult) error {
	// Look for any element with Address and Name attributes
	type xmlAny struct {
		Address string `xml:"Address,attr"`
		Name    string `xml:"Name,attr"`
		DPT     string `xml:"DatapointType,attr"`
		DPTAlt  string `xml:"DPT,attr"`
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidFile, err)
		}

		if se, ok := token.(xml.StartElement); ok {
			var elem xmlAny
			if err := decoder.DecodeElement(&elem, &se); err != nil {
				continue
			}

			if elem.Address != "" && isValidGA(elem.Address) {
				dpt := elem.DPT
				if dpt == "" {
					dpt = elem.DPTAlt
				}
				result.UnmappedAddresses = append(result.UnmappedAddresses, GroupAddress{
					Address: normaliseGA(elem.Address),
					Name:    elem.Name,
					DPT:     normaliseDPT(dpt),
				})
			}
		}
	}

	if len(result.UnmappedAddresses) == 0 {
		return ErrNoGroupAddresses
	}

	return nil
}

// parseXML parses a standalone XML export.
func (p *Parser) parseXML(data []byte, result *ParseResult) error {
	// Try parsing as GroupAddresses.xml format first
	if err := p.parseGroupAddressesXML(data, result); err == nil {
		return nil
	}

	// Fall back to generic XML parsing
	return p.parseGenericXML(data, result)
}

// parseCSV parses a CSV export of group addresses.
func (p *Parser) parseCSV(data []byte, result *ParseResult) error {
	lines := strings.Split(string(data), "\n")
	if len(lines) < 2 {
		return ErrNoGroupAddresses
	}

	// Parse header to find column indices
	header := parseCSVLine(lines[0])
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(strings.Trim(col, "\""))] = i
	}

	// Find address column
	addrCol := -1
	for _, name := range []string{"address", "groupaddress", "group address", "ga"} {
		if idx, ok := colIndex[name]; ok {
			addrCol = idx
			break
		}
	}
	if addrCol == -1 {
		return ErrInvalidFile
	}

	// Find other columns
	nameCol := findColumn(colIndex, "name", "description", "bezeichnung")
	dptCol := findColumn(colIndex, "datapointtype", "dpt", "datapoint")

	// Parse data rows
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := parseCSVLine(line)
		if len(fields) <= addrCol {
			continue
		}

		addr := strings.Trim(fields[addrCol], "\"")
		if !isValidGA(addr) {
			continue
		}

		ga := GroupAddress{Address: normaliseGA(addr)}
		if nameCol >= 0 && nameCol < len(fields) {
			ga.Name = strings.Trim(fields[nameCol], "\"")
		}
		if dptCol >= 0 && dptCol < len(fields) {
			ga.DPT = normaliseDPT(strings.Trim(fields[dptCol], "\""))
		}

		result.UnmappedAddresses = append(result.UnmappedAddresses, ga)
	}

	if len(result.UnmappedAddresses) == 0 {
		return ErrNoGroupAddresses
	}

	return nil
}

// locationNode is an internal tree node used to build the location hierarchy.
type locationNode struct {
	name     string
	children map[string]*locationNode
	hasGAs   bool // true if GAs exist directly under this node
}

// domainNames contains common ETS top-level functional groupings (EN + DE)
// that represent logical domains, not physical locations. These are skipped
// when building the location hierarchy.
var domainNames = map[string]bool{
	"lighting": true, "beleuchtung": true, "licht": true,
	"blinds": true, "jalousie": true, "jalousien": true,
	"shutter": true, "shutters": true, "rolladen": true,
	"hvac": true, "climate": true, "klima": true,
	"heating": true, "heizung": true, "cooling": true,
	"sensors": true, "sensoren": true,
	"scenes": true, "szenen": true,
	"security": true, "sicherheit": true,
	"energy": true, "energie": true,
	"audio": true, "video": true, "media": true,
}

// extractLocations builds structured Location objects from the hierarchy paths
// already captured on each GroupAddress and DetectedDevice during parsing.
// It deduplicates by slug, filters out domain-level names, and classifies
// nodes as areas (floors) or rooms based on tree position.
func (p *Parser) extractLocations(result *ParseResult) {
	// 1. Collect all unique location paths
	pathSet := make(map[string]bool)
	for _, ga := range result.UnmappedAddresses {
		if ga.Location != "" {
			pathSet[ga.Location] = true
		}
	}
	for _, dev := range result.Devices {
		if dev.SourceLocation != "" {
			pathSet[dev.SourceLocation] = true
		}
		for _, addr := range dev.Addresses {
			if addr.Description != "" {
				// Description doesn't carry location — skip
				continue
			}
		}
	}

	if len(pathSet) == 0 {
		return
	}

	// 2. Build a tree from path segments
	root := &locationNode{children: make(map[string]*locationNode)}

	for path := range pathSet {
		parts := strings.Split(path, " > ")
		node := root
		for i, part := range parts {
			if _, ok := node.children[part]; !ok {
				node.children[part] = &locationNode{
					name:     part,
					children: make(map[string]*locationNode),
				}
			}
			child := node.children[part]
			// Last segment = has GAs directly under it
			if i == len(parts)-1 {
				child.hasGAs = true
			}
			node = child
		}
	}

	// 3. Walk the tree, classify nodes, skip domain names
	seen := make(map[string]bool) // dedup by slug

	var walk func(node *locationNode, parentID string, depth int)
	walk = func(node *locationNode, parentID string, depth int) {
		for _, child := range node.children {
			slug := generateSlug(child.name)

			// Skip domain-like names at the top level
			if depth == 0 && domainNames[strings.ToLower(child.name)] {
				// Still recurse to find physical location nodes underneath
				walk(child, parentID, depth+1)
				continue
			}

			// Skip if already seen (dedup across domains)
			if seen[slug] {
				walk(child, slug, depth+1)
				continue
			}
			seen[slug] = true

			loc := Location{
				ID:       slug,
				Name:     child.name,
				ParentID: parentID,
			}

			if child.hasGAs && len(child.children) == 0 {
				// Leaf node with GAs = room
				loc.Type = "room"
				loc.SuggestedRoomID = slug
				if parentID != "" {
					loc.SuggestedAreaID = parentID
				}
			} else {
				// Non-leaf or intermediate node = area/floor
				loc.Type = "floor"
				loc.SuggestedAreaID = slug
			}

			result.Locations = append(result.Locations, loc)
			walk(child, slug, depth+1)
		}
	}

	walk(root, "", 0)

	// 4. Sort: areas before rooms so createLocationsFromETS can create parents first
	sort.SliceStable(result.Locations, func(i, j int) bool {
		iIsArea := result.Locations[i].Type != "room" && result.Locations[i].Type != "space"
		jIsArea := result.Locations[j].Type != "room" && result.Locations[j].Type != "space"
		if iIsArea != jIsArea {
			return iIsArea
		}
		return false
	})
}

// detectDevices groups addresses into logical devices using detection rules.
func (p *Parser) detectDevices(result *ParseResult) {
	// Group addresses by name prefix
	groups := p.groupByNamePrefix(result.UnmappedAddresses)

	var mapped []GroupAddress
	for prefix, addresses := range groups {
		device := p.tryDetectDevice(prefix, addresses)
		if device != nil {
			result.Devices = append(result.Devices, *device)
			// Mark these addresses as mapped
			for _, addr := range device.Addresses {
				mapped = append(mapped, GroupAddress{Address: addr.GA})
			}
		}
	}

	// Remove mapped addresses from unmapped list
	mappedSet := make(map[string]bool)
	for _, ga := range mapped {
		mappedSet[ga.Address] = true
	}

	var stillUnmapped []GroupAddress
	for _, ga := range result.UnmappedAddresses {
		if !mappedSet[ga.Address] {
			ga.Reason = "No matching device pattern"
			stillUnmapped = append(stillUnmapped, ga)
		}
	}
	result.UnmappedAddresses = stillUnmapped
}

// groupByNamePrefix groups addresses by their name prefix.
func (p *Parser) groupByNamePrefix(addresses []GroupAddress) map[string][]GroupAddress {
	groups := make(map[string][]GroupAddress)

	for _, addr := range addresses {
		prefix := extractNamePrefix(addr.Name)
		if prefix == "" {
			prefix = addr.Address // Use address as fallback
		}
		groups[prefix] = append(groups[prefix], addr)
	}

	return groups
}

// tryDetectDevice attempts to detect a device from a group of addresses.
func (p *Parser) tryDetectDevice(prefix string, addresses []GroupAddress) *DetectedDevice {
	// Try each detection rule in priority order
	for _, rule := range p.detectionRules {
		if device := rule.TryMatch(prefix, addresses); device != nil {
			return device
		}
	}

	// If we have only one address, create a simple device
	if len(addresses) == 1 {
		addr := addresses[0]
		device := &DetectedDevice{
			SuggestedID:     generateSlug(prefix),
			SuggestedName:   cleanName(prefix),
			DetectedType:    inferTypeFromDPT(addr.DPT),
			Confidence:      0.4, // Low confidence for single-address devices
			SuggestedDomain: inferDomainFromDPT(addr.DPT),
			SourceLocation:  addr.Location,
			Addresses: []DeviceAddress{{
				GA:                addr.Address,
				Name:              addr.Name,
				DPT:               addr.DPT,
				SuggestedFunction: inferFunctionFromDPT(addr.DPT, addr.Name),
				SuggestedFlags:    inferFlags(addr.DPT, addr.Name),
				Description:       addr.Description,
			}},
		}

		if addr.Location != "" {
			device.SuggestedRoom = extractRoomFromLocation(addr.Location)
			device.SuggestedArea = extractAreaFromLocation(addr.Location)
		}

		return device
	}

	return nil
}

// calculateStatistics computes parse statistics.
func (p *Parser) calculateStatistics(result *ParseResult) {
	stats := ParseStatistics{
		TotalGroupAddresses: len(result.UnmappedAddresses),
		DetectedDevices:     len(result.Devices),
		UnmappedAddresses:   len(result.UnmappedAddresses),
	}

	for _, dev := range result.Devices {
		stats.TotalGroupAddresses += len(dev.Addresses)

		switch ConfidenceLevel(dev.Confidence) {
		case "high":
			stats.HighConfidence++
		case "medium":
			stats.MediumConfidence++
		default:
			stats.LowConfidence++
		}
	}

	result.Statistics = stats
}

// Helper functions

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("opening zip file: %w", err)
	}
	defer rc.Close()

	// Limit read size
	data, err := io.ReadAll(io.LimitReader(rc, MaxFileSize))
	if err != nil {
		return nil, fmt.Errorf("reading zip file: %w", err)
	}
	return data, nil
}

func isZipFile(data []byte) bool {
	return len(data) >= 4 && data[0] == 0x50 && data[1] == 0x4B
}

func isXMLFile(data []byte) bool {
	trimmed := bytes.TrimLeftFunc(data, unicode.IsSpace)
	return bytes.HasPrefix(trimmed, []byte("<?xml")) || bytes.HasPrefix(trimmed, []byte("<"))
}

func generateImportID() string {
	b := make([]byte, importIDBytes)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read only fails on unsupported platforms (Plan 9 without /dev/random).
		// Fall back to timestamp-based ID for robustness.
		return "imp_" + hex.EncodeToString([]byte(time.Now().Format("20060102150405")))
	}
	return "imp_" + hex.EncodeToString(b)
}

// Precompiled regexes for DPT normalisation.
var (
	reDPTComplete = regexp.MustCompile(`^\d+\.\d{3}$`)
	reDPST        = regexp.MustCompile(`DPST-(\d+)-(\d+)`)
	reDPT         = regexp.MustCompile(`DPT-?(\d+)`)
	reDPTPartial  = regexp.MustCompile(`^(\d+)\.(\d+)$`)
)

// normaliseDPT converts ETS DPT format to standard format.
// DPST-1-1 -> 1.001, DPST-5-1 -> 5.001, etc.
func normaliseDPT(dpt string) string {
	if dpt == "" {
		return ""
	}

	// Already in correct format?
	if reDPTComplete.MatchString(dpt) {
		return dpt
	}

	// DPST-X-Y format
	if matches := reDPST.FindStringSubmatch(dpt); len(matches) == regexMatchCount3 {
		return fmt.Sprintf("%s.%03s", matches[1], matches[2])
	}

	// DPT-X format (main type only)
	if matches := reDPT.FindStringSubmatch(dpt); len(matches) == regexMatchCount2 {
		return matches[1] + ".001"
	}

	// Just numbers
	if matches := reDPTPartial.FindStringSubmatch(dpt); len(matches) == regexMatchCount3 {
		subtype := matches[2]
		for len(subtype) < 3 {
			subtype = "0" + subtype
		}
		return matches[1] + "." + subtype
	}

	return dpt
}

// isValidGA checks if a string is a valid group address format.
func isValidGA(addr string) bool {
	// Accept both 3-level (1/2/3) and 2-level (1/2) formats
	// Also accept integer format (will need conversion)
	re := regexp.MustCompile(`^(\d{1,2}/\d{1,2}/\d{1,3}|\d{1,2}/\d{1,4}|\d+)$`)
	return re.MatchString(addr)
}

// normaliseGA converts a group address to standard 3-level format (main/middle/sub).
// Handles: integer (2048), 2-level (1/0), and 3-level (1/0/0) formats.
func normaliseGA(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}

	// Already in 3-level format?
	if strings.Count(addr, "/") == 2 {
		return addr
	}

	// 2-level format (main/sub) -> convert to 3-level
	if strings.Count(addr, "/") == 1 {
		parts := strings.Split(addr, "/")
		if len(parts) == 2 {
			// 2-level: main (5 bits) / sub (11 bits)
			// Convert to 3-level: main (5 bits) / middle (3 bits) / sub (8 bits)
			main := parts[0]
			subInt := 0
			fmt.Sscanf(parts[1], "%d", &subInt)
			middle := (subInt >> 8) & 0x07
			sub := subInt & 0xFF
			return fmt.Sprintf("%s/%d/%d", main, middle, sub)
		}
	}

	// Integer format -> convert to 3-level
	var intVal int
	if _, err := fmt.Sscanf(addr, "%d", &intVal); err == nil && intVal >= 0 {
		// KNX GA integer encoding: main (5 bits) | middle (3 bits) | sub (8 bits)
		// Total 16 bits: MMMMM MMM SSSSSSSS
		main := (intVal >> 11) & 0x1F
		middle := (intVal >> 8) & 0x07
		sub := intVal & 0xFF
		return fmt.Sprintf("%d/%d/%d", main, middle, sub)
	}

	// Return as-is if we can't parse it
	return addr
}

func parseCSVLine(line string) []string {
	// Simple CSV parser (handles quoted fields)
	var fields []string
	var current strings.Builder
	inQuotes := false

	for _, r := range line {
		switch {
		case r == '"':
			inQuotes = !inQuotes
		case (r == ',' || r == '\t' || r == ';') && !inQuotes:
			fields = append(fields, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	fields = append(fields, current.String())

	return fields
}

func findColumn(index map[string]int, names ...string) int {
	for _, name := range names {
		if idx, ok := index[name]; ok {
			return idx
		}
	}
	return -1
}

func extractNamePrefix(name string) string {
	// Handle colon-separated suffixes first (e.g., "Ch-3 - Blinds Control : Step/stop")
	if colonIdx := strings.LastIndex(name, " : "); colonIdx > 0 {
		name = strings.TrimSpace(name[:colonIdx])
	} else if colonIdx := strings.LastIndex(name, ":"); colonIdx > 0 {
		name = strings.TrimSpace(name[:colonIdx])
	}

	// Remove common suffixes - ORDER MATTERS: longer/compound suffixes first!
	suffixes := []string{
		// Compound suffixes (must come first)
		" brightness status", " helligkeit status", " helligkeit rückmeldung",
		" position status", " position rückmeldung",
		" slat status", " lamelle status", " lamelle rückmeldung",
		" switch status", " schalten status", " schalten rückmeldung",
		" step/stop", " step-stop",
		" up/down", " auf/ab",
		" on/off", " ein/aus",
		// Single word suffixes
		" switch", " switching", " status", " feedback", " rückmeldung",
		" dimming", " brightness", " helligkeit", " level",
		" position", " move", " stop",
		" slat", " lamelle", " tilt", " angle",
		" schalten", " step", " stopp",
		" control", " steuerung", " cmd", " command",
	}

	lower := strings.ToLower(name)
	for _, suffix := range suffixes {
		if strings.HasSuffix(lower, suffix) {
			return strings.TrimSpace(name[:len(name)-len(suffix)])
		}
	}

	// Try to find common word boundary
	words := strings.Fields(name)
	if len(words) > 1 {
		// Keep all but last word if it looks like a function
		lastWord := strings.ToLower(words[len(words)-1])
		functionWords := map[string]bool{
			"switch": true, "dimming": true, "status": true, "brightness": true,
			"position": true, "move": true, "stop": true, "value": true,
			"schalten": true, "dimmen": true, "wert": true,
			"control": true, "steuerung": true,
		}
		if functionWords[lastWord] {
			return strings.Join(words[:len(words)-1], " ")
		}
	}

	return name
}

func generateSlug(name string) string {
	// Convert to lowercase, replace spaces/special chars with hyphens
	slug := strings.ToLower(name)
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	if slug == "" {
		slug = "device"
	}

	return slug
}

func cleanName(name string) string {
	// Title case and clean up
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

func inferTypeFromDPT(dpt string) string {
	if dpt == "" {
		return "unknown"
	}

	// Check specific DPT subtypes first (per KNX Standard v3.0.0)
	switch dpt {
	// Lighting
	case "1.001": // Switch
		return "light_switch"
	case "5.001": // Scaling/Brightness
		return "light_dimmer"
	case "7.600": // Color temperature Kelvin
		return "light_ct"
	case "232.600": // RGB
		return "light_rgb"
	case "251.600": // RGBW
		return "light_rgbw"

	// Blinds/Shutters
	case "1.008": // Up/Down
		return "blind_position"
	case "1.009": // Open/Close (window/door)
		return "window_sensor"
	case "1.017": // Trigger
		return "blind_position"

	// Climate sensors
	case "9.001": // Temperature °C
		return "temperature_sensor"
	case "9.007": // Humidity %
		return "humidity_sensor"
	case "9.004": // Lux
		return "brightness_sensor"
	case "9.005": // Wind speed m/s
		return "wind_sensor"

	// Presence/Motion
	case "1.002": // Bool (motion)
		return "motion_sensor"
	case "1.018": // Occupancy
		return "presence_sensor"

	// Scenes
	case "17.001", "18.001": // Scene number/control
		return "scene_controller"

	// HVAC
	case "20.102": // HVAC mode
		return "thermostat"
	}

	// Fall back to main type inference
	mainType := strings.Split(dpt, ".")[0]
	switch mainType {
	case "1":
		return typeSwitch
	case "3": // Dimming control
		return "light_dimmer"
	case "5": // Unsigned 8-bit (often brightness)
		return "light_dimmer"
	case "9": // 2-byte float (sensors)
		return typeSensor
	case "13": // 4-byte signed (energy counters)
		return "energy_meter"
	case "14": // 4-byte float (power, voltage, current)
		return "energy_meter"
	case "20": // HVAC enum
		return "thermostat"
	default:
		return "unknown"
	}
}

func inferDomainFromDPT(dpt string) string {
	if dpt == "" {
		return "sensor"
	}

	// Check specific DPTs first (per KNX Standard v3.0.0)
	switch dpt {
	// Lighting
	case "1.001", "5.001", "7.600", "232.600", "251.600":
		return "lighting"
	// Blinds
	case "1.008", "1.017":
		return "blinds"
	// Climate
	case "9.001", "9.007", "20.102":
		return "climate"
	// Security/access
	case "1.009": // Open/close
		return "security"
	// Scenes
	case "17.001", "18.001":
		return "scene"
	}

	// Fall back to main type inference
	mainType := strings.Split(dpt, ".")[0]
	switch mainType {
	case "1": // Boolean - context dependent, default to lighting
		return "lighting"
	case "3", "5": // Dimming control, scaling
		return "lighting"
	case "9": // 2-byte float - usually sensors
		return "sensor"
	case "13", "14": // Energy counters, power measurements
		return "energy"
	case "17", "18": // Scene
		return "scene"
	case "20": // HVAC enum
		return "climate"
	default:
		return "sensor"
	}
}

func inferFunctionFromDPT(dpt, name string) string {
	// Check name first for function hints
	if fn := inferFunctionFromName(name); fn != "" {
		return fn
	}

	// Fall back to DPT-based inference
	return inferFunctionFromDPTValue(dpt)
}

func inferFunctionFromName(name string) string {
	nameLower := strings.ToLower(name)

	// Name pattern to function mapping (English + German terms from ETS)
	namePatterns := []struct {
		keywords []string
		function string
	}{
		// Switching
		{[]string{"switch", "schalten", "on/off", "ein/aus"}, typeSwitch},
		{[]string{"status", "feedback", "rückmeldung", "state", "zustand"}, "switch_status"},

		// Dimming
		{[]string{"brightness", "dimm", "helligkeit", "level"}, "brightness"},
		{[]string{"brightness status", "helligkeit status"}, "brightness_status"},

		// Blinds/shutters
		{[]string{"position", "höhe", "height"}, "position"},
		{[]string{"slat", "lamelle", "tilt", "neigung", "angle", "winkel"}, "slat"},
		{[]string{"move", "fahren", "up/down", "auf/ab"}, "move"},
		{[]string{"stop", "stopp"}, "stop"},

		// Climate
		{[]string{"temperature", "temperatur", "temp"}, "temperature"},
		{[]string{"setpoint", "sollwert", "soll"}, "setpoint"},
		{[]string{"humidity", "feuchte", "luftfeuchte"}, "humidity"},
		{[]string{"co2", "kohlendioxid"}, "co2"},

		// Presence/motion
		{[]string{"presence", "präsenz", "anwesenheit"}, "presence"},
		{[]string{"motion", "bewegung"}, "motion"},
		{[]string{"occupancy", "belegung"}, "occupancy"},

		// Light sensors
		{[]string{"lux", "helligkeit", "brightness sensor"}, "lux"},

		// Wind/weather
		{[]string{"wind", "windgeschwindigkeit"}, "wind_speed"},
		{[]string{"rain", "regen"}, "rain"},

		// Scenes
		{[]string{"scene", "szene"}, "scene_number"},

		// Energy
		{[]string{"power", "leistung", "watt"}, "power"},
		{[]string{"energy", "energie", "verbrauch"}, "active_energy"},
		{[]string{"voltage", "spannung"}, "voltage"},
		{[]string{"current", "strom"}, "current"},
	}

	for _, p := range namePatterns {
		for _, kw := range p.keywords {
			if strings.Contains(nameLower, kw) {
				return p.function
			}
		}
	}
	return ""
}

func inferFunctionFromDPTValue(dpt string) string {
	if dpt == "" {
		return "value"
	}

	// DPT to function mapping (per KNX Standard v3.0.0)
	dptFunctions := map[string]string{
		// Boolean/switching
		"1.001": typeSwitch,
		"1.002": "presence",
		"1.003": "enable",
		"1.008": "move",
		"1.009": "open_close",
		"1.010": "start_stop",
		"1.017": "trigger",
		"1.018": "occupancy",

		// Dimming control
		"3.007": "dimming_control",
		"3.008": "blind_control",

		// Scaling/position
		"5.001": "brightness",
		"5.003": "angle",
		"5.004": "percentage",

		// 2-byte float sensors
		"9.001": "temperature",
		"9.002": "temperature_difference",
		"9.004": "lux",
		"9.005": "wind_speed",
		"9.007": "humidity",
		"9.008": "co2",
		"9.020": "voltage",
		"9.021": "current",

		// Color
		"7.600":   "color_temperature",
		"232.600": "rgb",
		"251.600": "rgbw",

		// Energy
		"13.010": "active_energy",
		"13.013": "active_energy_kwh",
		"14.019": "current",
		"14.027": "voltage",
		"14.056": "power",

		// Scene
		"17.001": "scene_number",
		"18.001": "scene_control",

		// HVAC
		"20.102": "hvac_mode",
	}

	if fn, ok := dptFunctions[dpt]; ok {
		return fn
	}
	return "value"
}

func inferFlags(dpt, name string) []string {
	nameLower := strings.ToLower(name)

	// Status/feedback addresses are typically read + transmit
	if strings.Contains(nameLower, "status") || strings.Contains(nameLower, "feedback") || strings.Contains(nameLower, "rückmeldung") {
		return []string{"read", "transmit"}
	}

	// Sensor values are typically read + transmit
	mainType := strings.Split(dpt, ".")[0]
	if mainType == "9" || mainType == "14" {
		return []string{"read", "transmit"}
	}

	// Default to write for commands
	return []string{"write"}
}

func extractRoomFromLocation(location string) string {
	parts := strings.Split(location, " > ")
	if len(parts) >= 2 {
		// Last part is often the room
		return generateSlug(parts[len(parts)-1])
	}
	return ""
}

func extractAreaFromLocation(location string) string {
	parts := strings.Split(location, " > ")
	if len(parts) >= 2 {
		// Second part is often the floor/area
		return generateSlug(parts[1])
	}
	if len(parts) == 1 {
		return generateSlug(parts[0])
	}
	return ""
}

func extractETSVersion(data []byte) string {
	re := regexp.MustCompile(`ToolVersion="([^"]+)"`)
	if matches := re.FindSubmatch(data); len(matches) == 2 {
		return string(matches[1])
	}
	return ""
}
