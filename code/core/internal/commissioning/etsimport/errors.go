package etsimport

import "errors"

// Sentinel errors for ETS import operations.
var (
	// ErrInvalidFile indicates the file is not a valid ETS export.
	ErrInvalidFile = errors.New("invalid ETS project file")

	// ErrUnsupportedVersion indicates an unsupported ETS version.
	ErrUnsupportedVersion = errors.New("unsupported ETS version")

	// ErrCorruptArchive indicates the ZIP archive is corrupted.
	ErrCorruptArchive = errors.New("corrupt archive")

	// ErrNoGroupAddresses indicates no group addresses were found.
	ErrNoGroupAddresses = errors.New("no group addresses found in project")

	// ErrEncodingError indicates a character encoding issue.
	ErrEncodingError = errors.New("encoding error")

	// ErrFileTooLarge indicates the file exceeds the size limit.
	ErrFileTooLarge = errors.New("file exceeds maximum size limit")

	// ErrParseTimeout indicates parsing took too long.
	ErrParseTimeout = errors.New("parse timeout exceeded")

	// ErrImportNotFound indicates the import ID was not found.
	ErrImportNotFound = errors.New("import session not found")

	// ErrImportExpired indicates the import session has expired.
	ErrImportExpired = errors.New("import session expired")
)

// Warning codes for non-fatal parse issues.
const (
	WarnRoomNotFound     = "ROOM_NOT_FOUND"
	WarnDPTUnknown       = "DPT_UNKNOWN"
	WarnDuplicateGA      = "DUPLICATE_GA"
	WarnDeviceConflict   = "DEVICE_CONFLICT"
	WarnLowConfidence    = "LOW_CONFIDENCE"
	WarnMissingDPT       = "MISSING_DPT"
	WarnNameTruncated    = "NAME_TRUNCATED"
	WarnLocationAmbigous = "LOCATION_AMBIGUOUS"
)
