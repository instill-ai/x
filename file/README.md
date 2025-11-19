# File Utilities Package

The `file` package provides centralized utilities for file type detection, MIME type conversion, and file format handling across Instill AI backends.

## Features

- **File Type Detection**: Automatic detection from MIME types and file extensions
- **MIME Type Mapping**: Convert between File_Type enums and MIME type strings
- **Media Type Categorization**: Group files into document, image, audio, and video categories
- **Format Conversion Detection**: Determine if files need conversion to AI-supported formats
- **Data URI Generation**: Create data URI prefixes for base64-encoded content
- **Extension Mapping**: Convert between file types and extensions

## Installation

```go
import "github.com/instill-ai/x/file"
```

## Functions

### FileTypeToMimeType

Convert a `File_Type` enum to its corresponding MIME type string.

```go
mimeType := file.FileTypeToMimeType(artifactpb.File_TYPE_PDF)
// Returns: "application/pdf"
```

### DetermineFileType

Detect file type from MIME type (content-type) and filename. Checks MIME type first, then falls back to file extension.

```go
fileType := file.DetermineFileType("application/pdf", "document.bin")
// Returns: artifactpb.File_TYPE_PDF

fileType = file.DetermineFileType("", "document.docx")
// Returns: artifactpb.File_TYPE_DOCX
```

### FileTypeToMediaType

Map a `File_Type` to its broader `File_FileMediaType` category (document, image, audio, video).

```go
mediaType := file.FileTypeToMediaType(artifactpb.File_TYPE_PDF)
// Returns: artifactpb.File_FILE_MEDIA_TYPE_DOCUMENT
```

### GetFileMediaType

Determine media type from `File_Type` with MIME type fallback.

```go
mediaType := file.GetFileMediaType(artifactpb.File_TYPE_UNSPECIFIED, "image/custom")
// Returns: artifactpb.File_FILE_MEDIA_TYPE_IMAGE
```

### FormatToFileType

Convert a format string (e.g., "pdf", "png") to `File_Type` enum.

```go
fileType := file.FormatToFileType("pdf")
// Returns: artifactpb.File_TYPE_PDF

fileType = file.FormatToFileType(".png")
// Returns: artifactpb.File_TYPE_PNG
```

### ConvertFileTypeString

Convert database file type string to `File_Type` enum. Supports both "TYPE_*" and "FILE_TYPE_*" formats.

```go
fileType := file.ConvertFileTypeString("TYPE_PDF")
// Returns: artifactpb.File_TYPE_PDF

fileType = file.ConvertFileTypeString("FILE_TYPE_PDF")
// Returns: artifactpb.File_TYPE_PDF
```

### GetDataURIPrefix

Get data URI prefix (with MIME type) for a file type.

```go
prefix := file.GetDataURIPrefix(artifactpb.File_TYPE_PDF)
// Returns: "data:application/pdf;base64,"
```

### NeedsFileTypeConversion

Check if a file needs conversion to AI-supported formats. Returns whether conversion is needed, the target format, and the target file type.

```go
needsConv, format, targetType := file.NeedsFileTypeConversion(artifactpb.File_TYPE_DOCX)
// Returns: true, "pdf", artifactpb.File_TYPE_PDF

needsConv, _, _ = file.NeedsFileTypeConversion(artifactpb.File_TYPE_PDF)
// Returns: false, "", artifactpb.File_TYPE_UNSPECIFIED
```

### FileTypeToExtension

Get the standard file extension for a file type (without dot).

```go
ext := file.FileTypeToExtension(artifactpb.File_TYPE_PDF)
// Returns: "pdf"
```

## AI-Supported Standard Formats

For optimal AI/LLM processing, the package recognizes these standard formats:

- **Documents**: PDF (converts from DOC, DOCX, PPT, PPTX, XLS, XLSX, HTML, TEXT, MARKDOWN, CSV)
- **Images**: PNG (converts from JPEG, GIF, BMP, TIFF, AVIF, WEBP, HEIC, HEIF)
- **Audio**: OGG (converts from MP3, WAV, AAC, M4A, WMA, FLAC, AIFF, WEBM_AUDIO)
- **Video**: MP4 (converts from MKV, MPEG, MOV, AVI, FLV, WMV, WEBM_VIDEO)

## Usage Examples

### File Upload Processing

```go
// Detect file type from upload
contentType := req.Header.Get("Content-Type")
fileName := req.FormValue("filename")
fileType := file.DetermineFileType(contentType, fileName)

// Get MIME type for storage
mimeType := file.FileTypeToMimeType(fileType)

// Check if conversion is needed
if needsConv, format, targetType := file.NeedsFileTypeConversion(fileType); needsConv {
    // Convert file to target format
    convertedFile := convertToFormat(uploadedFile, format)
    // Use targetType for converted file
}
```

### Data URI Generation

```go
// Generate data URI for base64-encoded content
prefix := file.GetDataURIPrefix(artifactpb.File_TYPE_PNG)
dataURI := prefix + base64EncodedContent
// Result: "data:image/png;base64,iVBORw0KGgo..."
```

### Database File Type Conversion

```go
// Convert database enum string to File_Type
dbFileType := "TYPE_PDF" // Database format (enforced by CHECK constraint)
fileType := file.ConvertFileTypeString(dbFileType)
```

## Benefits

- **Single Source of Truth**: All file type logic in one place
- **Consistency**: Same behavior across all backends (artifact-backend, agent-backend, etc.)
- **Maintainability**: Update once, apply everywhere
- **Testability**: Comprehensive test coverage
- **Performance**: No external dependencies, fast lookups

## Testing

Run tests with:

```bash
cd x/file
go test -v
```

All functions have comprehensive test coverage including edge cases.
