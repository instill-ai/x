package file

import (
	"path/filepath"
	"strings"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
)

// FileTypeToMimeType converts a File_Type enum to its corresponding MIME type string.
// This is useful for HTTP headers, multimodal AI processing, and file downloads.
func FileTypeToMimeType(fileType artifactpb.File_Type) string {
	switch fileType {
	// Text-based document types
	case artifactpb.File_TYPE_TEXT:
		return "text/plain"
	case artifactpb.File_TYPE_MARKDOWN:
		return "text/markdown"
	case artifactpb.File_TYPE_HTML:
		return "text/html"
	case artifactpb.File_TYPE_CSV:
		return "text/csv"

	// Container-based document types
	case artifactpb.File_TYPE_PDF:
		return "application/pdf"
	case artifactpb.File_TYPE_DOC:
		return "application/msword"
	case artifactpb.File_TYPE_DOCX:
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case artifactpb.File_TYPE_PPT:
		return "application/vnd.ms-powerpoint"
	case artifactpb.File_TYPE_PPTX:
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case artifactpb.File_TYPE_XLS:
		return "application/vnd.ms-excel"
	case artifactpb.File_TYPE_XLSX:
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

	// Image types
	case artifactpb.File_TYPE_PNG:
		return "image/png"
	case artifactpb.File_TYPE_JPEG:
		return "image/jpeg"
	case artifactpb.File_TYPE_GIF:
		return "image/gif"
	case artifactpb.File_TYPE_WEBP:
		return "image/webp"
	case artifactpb.File_TYPE_TIFF:
		return "image/tiff"
	case artifactpb.File_TYPE_BMP:
		return "image/bmp"
	case artifactpb.File_TYPE_HEIC:
		return "image/heic"
	case artifactpb.File_TYPE_HEIF:
		return "image/heif"
	case artifactpb.File_TYPE_AVIF:
		return "image/avif"

	// Audio types
	case artifactpb.File_TYPE_MP3:
		return "audio/mpeg"
	case artifactpb.File_TYPE_WAV:
		return "audio/wav"
	case artifactpb.File_TYPE_AAC:
		return "audio/aac"
	case artifactpb.File_TYPE_OGG:
		return "audio/ogg"
	case artifactpb.File_TYPE_FLAC:
		return "audio/flac"
	case artifactpb.File_TYPE_M4A:
		return "audio/mp4"
	case artifactpb.File_TYPE_WMA:
		return "audio/x-ms-wma"
	case artifactpb.File_TYPE_AIFF:
		return "audio/aiff"
	case artifactpb.File_TYPE_WEBM_AUDIO:
		return "audio/webm"

	// Video types
	case artifactpb.File_TYPE_MP4:
		return "video/mp4"
	case artifactpb.File_TYPE_MOV:
		return "video/quicktime"
	case artifactpb.File_TYPE_AVI:
		return "video/x-msvideo"
	case artifactpb.File_TYPE_MKV:
		return "video/x-matroska"
	case artifactpb.File_TYPE_WEBM_VIDEO:
		return "video/webm"
	case artifactpb.File_TYPE_FLV:
		return "video/x-flv"
	case artifactpb.File_TYPE_WMV:
		return "video/x-ms-wmv"
	case artifactpb.File_TYPE_MPEG:
		return "video/mpeg"

	default:
		return "application/octet-stream"
	}
}

// DetermineFileType detects the file type from MIME type (content-type) and filename.
// It first checks the MIME type, then falls back to file extension if needed.
func DetermineFileType(contentType, fileName string) artifactpb.File_Type {
	// Normalize content type to lowercase
	contentType = strings.ToLower(strings.TrimSpace(contentType))

	// Check MIME type first
	switch {
	// Documents
	case strings.Contains(contentType, "pdf"):
		return artifactpb.File_TYPE_PDF
	case strings.Contains(contentType, "application/vnd.openxmlformats-officedocument.wordprocessingml.document"):
		return artifactpb.File_TYPE_DOCX
	case strings.Contains(contentType, "application/msword"):
		return artifactpb.File_TYPE_DOC
	case strings.Contains(contentType, "application/vnd.openxmlformats-officedocument.presentationml.presentation"):
		return artifactpb.File_TYPE_PPTX
	case strings.Contains(contentType, "application/vnd.ms-powerpoint"):
		return artifactpb.File_TYPE_PPT
	case strings.Contains(contentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"):
		return artifactpb.File_TYPE_XLSX
	case strings.Contains(contentType, "application/vnd.ms-excel"):
		return artifactpb.File_TYPE_XLS
	case strings.Contains(contentType, "text/csv"):
		return artifactpb.File_TYPE_CSV
	case strings.Contains(contentType, "text/html"):
		return artifactpb.File_TYPE_HTML
	case strings.Contains(contentType, "text/plain"):
		return artifactpb.File_TYPE_TEXT
	case strings.Contains(contentType, "text/markdown"):
		return artifactpb.File_TYPE_MARKDOWN

	// Images
	case strings.Contains(contentType, "image/png"):
		return artifactpb.File_TYPE_PNG
	case strings.Contains(contentType, "image/jpeg"):
		return artifactpb.File_TYPE_JPEG
	case strings.Contains(contentType, "image/gif"):
		return artifactpb.File_TYPE_GIF
	case strings.Contains(contentType, "image/webp"):
		return artifactpb.File_TYPE_WEBP
	case strings.Contains(contentType, "image/tiff"):
		return artifactpb.File_TYPE_TIFF
	case strings.Contains(contentType, "image/bmp"):
		return artifactpb.File_TYPE_BMP
	case strings.Contains(contentType, "image/heic"):
		return artifactpb.File_TYPE_HEIC
	case strings.Contains(contentType, "image/heif"):
		return artifactpb.File_TYPE_HEIF
	case strings.Contains(contentType, "image/avif"):
		return artifactpb.File_TYPE_AVIF

	// Audio
	case strings.Contains(contentType, "audio/mpeg"):
		return artifactpb.File_TYPE_MP3
	case strings.Contains(contentType, "audio/wav"):
		return artifactpb.File_TYPE_WAV
	case strings.Contains(contentType, "audio/aac"):
		return artifactpb.File_TYPE_AAC
	case strings.Contains(contentType, "audio/ogg"):
		return artifactpb.File_TYPE_OGG
	case strings.Contains(contentType, "audio/flac"):
		return artifactpb.File_TYPE_FLAC
	case strings.Contains(contentType, "audio/mp4"):
		return artifactpb.File_TYPE_M4A
	case strings.Contains(contentType, "audio/x-ms-wma"):
		return artifactpb.File_TYPE_WMA
	case strings.Contains(contentType, "audio/aiff"):
		return artifactpb.File_TYPE_AIFF
	case strings.Contains(contentType, "audio/webm"):
		return artifactpb.File_TYPE_WEBM_AUDIO

	// Video
	case strings.Contains(contentType, "video/mp4"):
		return artifactpb.File_TYPE_MP4
	case strings.Contains(contentType, "video/quicktime"):
		return artifactpb.File_TYPE_MOV
	case strings.Contains(contentType, "video/x-msvideo"):
		return artifactpb.File_TYPE_AVI
	case strings.Contains(contentType, "video/x-matroska"):
		return artifactpb.File_TYPE_MKV
	case strings.Contains(contentType, "video/webm"):
		return artifactpb.File_TYPE_WEBM_VIDEO
	case strings.Contains(contentType, "video/x-flv"):
		return artifactpb.File_TYPE_FLV
	case strings.Contains(contentType, "video/x-ms-wmv"):
		return artifactpb.File_TYPE_WMV
	case strings.Contains(contentType, "video/mpeg"):
		return artifactpb.File_TYPE_MPEG
	}

	// Fallback to extension-based detection
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	// Documents
	case ".pdf":
		return artifactpb.File_TYPE_PDF
	case ".docx":
		return artifactpb.File_TYPE_DOCX
	case ".doc":
		return artifactpb.File_TYPE_DOC
	case ".pptx":
		return artifactpb.File_TYPE_PPTX
	case ".ppt":
		return artifactpb.File_TYPE_PPT
	case ".xlsx":
		return artifactpb.File_TYPE_XLSX
	case ".xls":
		return artifactpb.File_TYPE_XLS
	case ".csv":
		return artifactpb.File_TYPE_CSV
	case ".html", ".htm":
		return artifactpb.File_TYPE_HTML
	case ".txt":
		return artifactpb.File_TYPE_TEXT
	case ".md", ".markdown":
		return artifactpb.File_TYPE_MARKDOWN

	// Images
	case ".png":
		return artifactpb.File_TYPE_PNG
	case ".jpg", ".jpeg":
		return artifactpb.File_TYPE_JPEG
	case ".gif":
		return artifactpb.File_TYPE_GIF
	case ".webp":
		return artifactpb.File_TYPE_WEBP
	case ".tiff", ".tif":
		return artifactpb.File_TYPE_TIFF
	case ".bmp":
		return artifactpb.File_TYPE_BMP
	case ".heic":
		return artifactpb.File_TYPE_HEIC
	case ".heif":
		return artifactpb.File_TYPE_HEIF
	case ".avif":
		return artifactpb.File_TYPE_AVIF

	// Audio
	case ".mp3":
		return artifactpb.File_TYPE_MP3
	case ".wav":
		return artifactpb.File_TYPE_WAV
	case ".aac":
		return artifactpb.File_TYPE_AAC
	case ".ogg":
		return artifactpb.File_TYPE_OGG
	case ".flac":
		return artifactpb.File_TYPE_FLAC
	case ".m4a":
		return artifactpb.File_TYPE_M4A
	case ".wma":
		return artifactpb.File_TYPE_WMA
	case ".aiff", ".aif":
		return artifactpb.File_TYPE_AIFF

	// Video
	case ".mp4":
		return artifactpb.File_TYPE_MP4
	case ".mov":
		return artifactpb.File_TYPE_MOV
	case ".avi":
		return artifactpb.File_TYPE_AVI
	case ".mkv":
		return artifactpb.File_TYPE_MKV
	case ".webm":
		return artifactpb.File_TYPE_WEBM_VIDEO
	case ".flv":
		return artifactpb.File_TYPE_FLV
	case ".wmv":
		return artifactpb.File_TYPE_WMV
	case ".mpeg", ".mpg":
		return artifactpb.File_TYPE_MPEG

	default:
		return artifactpb.File_TYPE_UNSPECIFIED
	}
}

// FileTypeToMediaType maps File_Type to File_FileMediaType.
// This categorizes files into broader media categories (document, image, audio, video).
func FileTypeToMediaType(fileType artifactpb.File_Type) artifactpb.File_FileMediaType {
	switch fileType {
	// Document types
	case artifactpb.File_TYPE_PDF,
		artifactpb.File_TYPE_DOCX,
		artifactpb.File_TYPE_DOC,
		artifactpb.File_TYPE_PPTX,
		artifactpb.File_TYPE_PPT,
		artifactpb.File_TYPE_XLSX,
		artifactpb.File_TYPE_XLS,
		artifactpb.File_TYPE_CSV,
		artifactpb.File_TYPE_HTML,
		artifactpb.File_TYPE_TEXT,
		artifactpb.File_TYPE_MARKDOWN:
		return artifactpb.File_FILE_MEDIA_TYPE_DOCUMENT

	// Image types
	case artifactpb.File_TYPE_PNG,
		artifactpb.File_TYPE_JPEG,
		artifactpb.File_TYPE_GIF,
		artifactpb.File_TYPE_WEBP,
		artifactpb.File_TYPE_TIFF,
		artifactpb.File_TYPE_BMP,
		artifactpb.File_TYPE_HEIC,
		artifactpb.File_TYPE_HEIF,
		artifactpb.File_TYPE_AVIF:
		return artifactpb.File_FILE_MEDIA_TYPE_IMAGE

	// Audio types
	case artifactpb.File_TYPE_MP3,
		artifactpb.File_TYPE_WAV,
		artifactpb.File_TYPE_AAC,
		artifactpb.File_TYPE_OGG,
		artifactpb.File_TYPE_FLAC,
		artifactpb.File_TYPE_M4A,
		artifactpb.File_TYPE_WMA,
		artifactpb.File_TYPE_AIFF,
		artifactpb.File_TYPE_WEBM_AUDIO:
		return artifactpb.File_FILE_MEDIA_TYPE_AUDIO

	// Video types
	case artifactpb.File_TYPE_MP4,
		artifactpb.File_TYPE_MOV,
		artifactpb.File_TYPE_AVI,
		artifactpb.File_TYPE_MKV,
		artifactpb.File_TYPE_WEBM_VIDEO,
		artifactpb.File_TYPE_FLV,
		artifactpb.File_TYPE_WMV,
		artifactpb.File_TYPE_MPEG:
		return artifactpb.File_FILE_MEDIA_TYPE_VIDEO

	default:
		return artifactpb.File_FILE_MEDIA_TYPE_UNSPECIFIED
	}
}

// GetFileMediaType determines the FileMediaType from FileType and MIME type with fallback.
// First tries to get media type from FileType, then falls back to MIME type patterns.
func GetFileMediaType(fileType artifactpb.File_Type, mimeType string) artifactpb.File_FileMediaType {
	// First try to get media type from FileType
	mediaType := FileTypeToMediaType(fileType)
	if mediaType != artifactpb.File_FILE_MEDIA_TYPE_UNSPECIFIED {
		return mediaType
	}

	// Normalize MIME type to lowercase
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))

	// Check MIME type patterns for fallback detection
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return artifactpb.File_FILE_MEDIA_TYPE_IMAGE
	case strings.HasPrefix(mimeType, "audio/"):
		return artifactpb.File_FILE_MEDIA_TYPE_AUDIO
	case strings.HasPrefix(mimeType, "video/"):
		return artifactpb.File_FILE_MEDIA_TYPE_VIDEO
	case strings.HasPrefix(mimeType, "text/"):
		return artifactpb.File_FILE_MEDIA_TYPE_DOCUMENT
	case strings.HasPrefix(mimeType, "application/pdf"):
		return artifactpb.File_FILE_MEDIA_TYPE_DOCUMENT
	case strings.Contains(mimeType, "document"),
		strings.Contains(mimeType, "word"),
		strings.Contains(mimeType, "powerpoint"),
		strings.Contains(mimeType, "excel"):
		return artifactpb.File_FILE_MEDIA_TYPE_DOCUMENT
	}

	return artifactpb.File_FILE_MEDIA_TYPE_UNSPECIFIED
}

// FormatToFileType converts a format string (e.g., "pdf", "png", "mp4") to File_Type.
// This is useful when working with file conversion pipelines or AI processing.
func FormatToFileType(format string) artifactpb.File_Type {
	format = strings.ToLower(strings.TrimSpace(format))
	format = strings.TrimPrefix(format, ".")

	switch format {
	// Documents
	case "pdf":
		return artifactpb.File_TYPE_PDF
	case "docx":
		return artifactpb.File_TYPE_DOCX
	case "doc":
		return artifactpb.File_TYPE_DOC
	case "pptx":
		return artifactpb.File_TYPE_PPTX
	case "ppt":
		return artifactpb.File_TYPE_PPT
	case "xlsx":
		return artifactpb.File_TYPE_XLSX
	case "xls":
		return artifactpb.File_TYPE_XLS
	case "csv":
		return artifactpb.File_TYPE_CSV
	case "html", "htm":
		return artifactpb.File_TYPE_HTML
	case "txt":
		return artifactpb.File_TYPE_TEXT
	case "md", "markdown":
		return artifactpb.File_TYPE_MARKDOWN

	// Images
	case "png":
		return artifactpb.File_TYPE_PNG
	case "jpg", "jpeg":
		return artifactpb.File_TYPE_JPEG
	case "gif":
		return artifactpb.File_TYPE_GIF
	case "webp":
		return artifactpb.File_TYPE_WEBP
	case "tiff", "tif":
		return artifactpb.File_TYPE_TIFF
	case "bmp":
		return artifactpb.File_TYPE_BMP
	case "heic":
		return artifactpb.File_TYPE_HEIC
	case "heif":
		return artifactpb.File_TYPE_HEIF
	case "avif":
		return artifactpb.File_TYPE_AVIF

	// Audio
	case "mp3":
		return artifactpb.File_TYPE_MP3
	case "wav":
		return artifactpb.File_TYPE_WAV
	case "aac":
		return artifactpb.File_TYPE_AAC
	case "ogg":
		return artifactpb.File_TYPE_OGG
	case "flac":
		return artifactpb.File_TYPE_FLAC
	case "m4a":
		return artifactpb.File_TYPE_M4A
	case "wma":
		return artifactpb.File_TYPE_WMA
	case "aiff", "aif":
		return artifactpb.File_TYPE_AIFF

	// Video
	case "mp4":
		return artifactpb.File_TYPE_MP4
	case "mov":
		return artifactpb.File_TYPE_MOV
	case "avi":
		return artifactpb.File_TYPE_AVI
	case "mkv":
		return artifactpb.File_TYPE_MKV
	case "webm":
		return artifactpb.File_TYPE_WEBM_VIDEO
	case "flv":
		return artifactpb.File_TYPE_FLV
	case "wmv":
		return artifactpb.File_TYPE_WMV
	case "mpeg", "mpg":
		return artifactpb.File_TYPE_MPEG

	default:
		return artifactpb.File_TYPE_UNSPECIFIED
	}
}

// ConvertFileTypeString converts a database file type string to File_Type enum.
// Expects "TYPE_*" format (e.g., "TYPE_PDF", "TYPE_TEXT").
// Note: Legacy "FILE_TYPE_*" format was deprecated and removed in migration 000042.
func ConvertFileTypeString(dbType string) artifactpb.File_Type {
	dbType = strings.ToUpper(strings.TrimSpace(dbType))

	switch dbType {
	// Documents
	case "TYPE_PDF":
		return artifactpb.File_TYPE_PDF
	case "TYPE_DOCX":
		return artifactpb.File_TYPE_DOCX
	case "TYPE_DOC":
		return artifactpb.File_TYPE_DOC
	case "TYPE_PPTX":
		return artifactpb.File_TYPE_PPTX
	case "TYPE_PPT":
		return artifactpb.File_TYPE_PPT
	case "TYPE_XLSX":
		return artifactpb.File_TYPE_XLSX
	case "TYPE_XLS":
		return artifactpb.File_TYPE_XLS
	case "TYPE_CSV":
		return artifactpb.File_TYPE_CSV
	case "TYPE_HTML":
		return artifactpb.File_TYPE_HTML
	case "TYPE_TEXT":
		return artifactpb.File_TYPE_TEXT
	case "TYPE_MARKDOWN":
		return artifactpb.File_TYPE_MARKDOWN

	// Images
	case "TYPE_PNG":
		return artifactpb.File_TYPE_PNG
	case "TYPE_JPEG":
		return artifactpb.File_TYPE_JPEG
	case "TYPE_GIF":
		return artifactpb.File_TYPE_GIF
	case "TYPE_WEBP":
		return artifactpb.File_TYPE_WEBP
	case "TYPE_TIFF":
		return artifactpb.File_TYPE_TIFF
	case "TYPE_BMP":
		return artifactpb.File_TYPE_BMP
	case "TYPE_HEIC":
		return artifactpb.File_TYPE_HEIC
	case "TYPE_HEIF":
		return artifactpb.File_TYPE_HEIF
	case "TYPE_AVIF":
		return artifactpb.File_TYPE_AVIF

	// Audio
	case "TYPE_MP3":
		return artifactpb.File_TYPE_MP3
	case "TYPE_WAV":
		return artifactpb.File_TYPE_WAV
	case "TYPE_AAC":
		return artifactpb.File_TYPE_AAC
	case "TYPE_OGG":
		return artifactpb.File_TYPE_OGG
	case "TYPE_FLAC":
		return artifactpb.File_TYPE_FLAC
	case "TYPE_M4A":
		return artifactpb.File_TYPE_M4A
	case "TYPE_WMA":
		return artifactpb.File_TYPE_WMA
	case "TYPE_AIFF":
		return artifactpb.File_TYPE_AIFF
	case "TYPE_WEBM_AUDIO":
		return artifactpb.File_TYPE_WEBM_AUDIO

	// Video
	case "TYPE_MP4":
		return artifactpb.File_TYPE_MP4
	case "TYPE_MOV":
		return artifactpb.File_TYPE_MOV
	case "TYPE_AVI":
		return artifactpb.File_TYPE_AVI
	case "TYPE_MKV":
		return artifactpb.File_TYPE_MKV
	case "TYPE_WEBM_VIDEO":
		return artifactpb.File_TYPE_WEBM_VIDEO
	case "TYPE_FLV":
		return artifactpb.File_TYPE_FLV
	case "TYPE_WMV":
		return artifactpb.File_TYPE_WMV
	case "TYPE_MPEG":
		return artifactpb.File_TYPE_MPEG

	default:
		return artifactpb.File_TYPE_UNSPECIFIED
	}
}

// GetDataURIPrefix returns the data URI prefix (with MIME type) for a file type.
// Example: "data:application/pdf;base64,"
func GetDataURIPrefix(fileType artifactpb.File_Type) string {
	mimeType := FileTypeToMimeType(fileType)
	if mimeType == "application/octet-stream" {
		return ""
	}
	return "data:" + mimeType + ";base64,"
}

// NeedsFileTypeConversion checks if a file type needs conversion to AI-supported format.
// Returns (needsConversion bool, targetFormat string, targetFileType File_Type).
// Based on standard formats for AI/LLM processing: PNG (images), OGG (audio), MP4 (video), PDF (documents).
func NeedsFileTypeConversion(fileType artifactpb.File_Type) (needsConversion bool, targetFormat string, targetFileType artifactpb.File_Type) {
	switch fileType {
	// Standard image format - no conversion needed
	case artifactpb.File_TYPE_PNG:
		return false, "", artifactpb.File_TYPE_UNSPECIFIED

	// Convertible image formats - convert to PNG
	case artifactpb.File_TYPE_GIF,
		artifactpb.File_TYPE_BMP,
		artifactpb.File_TYPE_TIFF,
		artifactpb.File_TYPE_AVIF,
		artifactpb.File_TYPE_JPEG,
		artifactpb.File_TYPE_WEBP,
		artifactpb.File_TYPE_HEIC,
		artifactpb.File_TYPE_HEIF:
		return true, "png", artifactpb.File_TYPE_PNG

	// Standard audio format - no conversion needed
	case artifactpb.File_TYPE_OGG:
		return false, "", artifactpb.File_TYPE_UNSPECIFIED

	// Convertible audio formats - convert to OGG
	case artifactpb.File_TYPE_MP3,
		artifactpb.File_TYPE_WAV,
		artifactpb.File_TYPE_AAC,
		artifactpb.File_TYPE_M4A,
		artifactpb.File_TYPE_WMA,
		artifactpb.File_TYPE_FLAC,
		artifactpb.File_TYPE_AIFF,
		artifactpb.File_TYPE_WEBM_AUDIO:
		return true, "ogg", artifactpb.File_TYPE_OGG

	// Standard video format - no conversion needed
	case artifactpb.File_TYPE_MP4:
		return false, "", artifactpb.File_TYPE_UNSPECIFIED

	// Convertible video formats - convert to MP4
	case artifactpb.File_TYPE_MKV,
		artifactpb.File_TYPE_MPEG,
		artifactpb.File_TYPE_MOV,
		artifactpb.File_TYPE_AVI,
		artifactpb.File_TYPE_FLV,
		artifactpb.File_TYPE_WMV,
		artifactpb.File_TYPE_WEBM_VIDEO:
		return true, "mp4", artifactpb.File_TYPE_MP4

	// Standard document format - no conversion needed
	case artifactpb.File_TYPE_PDF:
		return false, "", artifactpb.File_TYPE_UNSPECIFIED

	// Convertible document formats - convert to PDF
	case artifactpb.File_TYPE_DOC,
		artifactpb.File_TYPE_DOCX,
		artifactpb.File_TYPE_PPT,
		artifactpb.File_TYPE_PPTX,
		artifactpb.File_TYPE_XLS,
		artifactpb.File_TYPE_XLSX,
		artifactpb.File_TYPE_HTML,
		artifactpb.File_TYPE_TEXT,
		artifactpb.File_TYPE_MARKDOWN,
		artifactpb.File_TYPE_CSV:
		return true, "pdf", artifactpb.File_TYPE_PDF

	default:
		return false, "", artifactpb.File_TYPE_UNSPECIFIED
	}
}

// GetConvertedFileTypeInfo returns the converted file type enum and extension for a given file type.
// This is used for determining standardized file conversions (e.g., DOCX → PDF).
// Returns (convertedFileType, extension, mimeType) or (UNSPECIFIED, "", "") if no conversion is defined.
func GetConvertedFileTypeInfo(fileType artifactpb.File_Type) (artifactpb.ConvertedFileType, string, string) {
	mediaType := FileTypeToMediaType(fileType)

	switch mediaType {
	case artifactpb.File_FILE_MEDIA_TYPE_DOCUMENT:
		return artifactpb.ConvertedFileType_CONVERTED_FILE_TYPE_DOCUMENT, "pdf", "application/pdf"
	case artifactpb.File_FILE_MEDIA_TYPE_IMAGE:
		return artifactpb.ConvertedFileType_CONVERTED_FILE_TYPE_IMAGE, "png", "image/png"
	case artifactpb.File_FILE_MEDIA_TYPE_AUDIO:
		return artifactpb.ConvertedFileType_CONVERTED_FILE_TYPE_AUDIO, "ogg", "audio/ogg"
	case artifactpb.File_FILE_MEDIA_TYPE_VIDEO:
		return artifactpb.ConvertedFileType_CONVERTED_FILE_TYPE_VIDEO, "mp4", "video/mp4"
	default:
		return artifactpb.ConvertedFileType_CONVERTED_FILE_TYPE_UNSPECIFIED, "", ""
	}
}

// FileTypeToExtension returns the standard file extension for a file type (without dot).
func FileTypeToExtension(fileType artifactpb.File_Type) string {
	switch fileType {
	// Documents
	case artifactpb.File_TYPE_PDF:
		return "pdf"
	case artifactpb.File_TYPE_DOCX:
		return "docx"
	case artifactpb.File_TYPE_DOC:
		return "doc"
	case artifactpb.File_TYPE_PPTX:
		return "pptx"
	case artifactpb.File_TYPE_PPT:
		return "ppt"
	case artifactpb.File_TYPE_XLSX:
		return "xlsx"
	case artifactpb.File_TYPE_XLS:
		return "xls"
	case artifactpb.File_TYPE_CSV:
		return "csv"
	case artifactpb.File_TYPE_HTML:
		return "html"
	case artifactpb.File_TYPE_TEXT:
		return "txt"
	case artifactpb.File_TYPE_MARKDOWN:
		return "md"

	// Images
	case artifactpb.File_TYPE_PNG:
		return "png"
	case artifactpb.File_TYPE_JPEG:
		return "jpg"
	case artifactpb.File_TYPE_GIF:
		return "gif"
	case artifactpb.File_TYPE_WEBP:
		return "webp"
	case artifactpb.File_TYPE_TIFF:
		return "tiff"
	case artifactpb.File_TYPE_BMP:
		return "bmp"
	case artifactpb.File_TYPE_HEIC:
		return "heic"
	case artifactpb.File_TYPE_HEIF:
		return "heif"
	case artifactpb.File_TYPE_AVIF:
		return "avif"

	// Audio
	case artifactpb.File_TYPE_MP3:
		return "mp3"
	case artifactpb.File_TYPE_WAV:
		return "wav"
	case artifactpb.File_TYPE_AAC:
		return "aac"
	case artifactpb.File_TYPE_OGG:
		return "ogg"
	case artifactpb.File_TYPE_FLAC:
		return "flac"
	case artifactpb.File_TYPE_M4A:
		return "m4a"
	case artifactpb.File_TYPE_WMA:
		return "wma"
	case artifactpb.File_TYPE_AIFF:
		return "aiff"

	// Video
	case artifactpb.File_TYPE_MP4:
		return "mp4"
	case artifactpb.File_TYPE_MOV:
		return "mov"
	case artifactpb.File_TYPE_AVI:
		return "avi"
	case artifactpb.File_TYPE_MKV:
		return "mkv"
	case artifactpb.File_TYPE_WEBM_VIDEO:
		return "webm"
	case artifactpb.File_TYPE_FLV:
		return "flv"
	case artifactpb.File_TYPE_WMV:
		return "wmv"
	case artifactpb.File_TYPE_MPEG:
		return "mpeg"

	default:
		return "bin"
	}
}
