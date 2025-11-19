package file

import (
	"testing"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
)

func TestFileTypeToMimeType(t *testing.T) {
	tests := []struct {
		name     string
		fileType artifactpb.File_Type
		want     string
	}{
		{"PDF", artifactpb.File_TYPE_PDF, "application/pdf"},
		{"DOCX", artifactpb.File_TYPE_DOCX, "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"PNG", artifactpb.File_TYPE_PNG, "image/png"},
		{"JPEG", artifactpb.File_TYPE_JPEG, "image/jpeg"},
		{"MP3", artifactpb.File_TYPE_MP3, "audio/mpeg"},
		{"MP4", artifactpb.File_TYPE_MP4, "video/mp4"},
		{"TEXT", artifactpb.File_TYPE_TEXT, "text/plain"},
		{"MARKDOWN", artifactpb.File_TYPE_MARKDOWN, "text/markdown"},
		{"UNSPECIFIED", artifactpb.File_TYPE_UNSPECIFIED, "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileTypeToMimeType(tt.fileType); got != tt.want {
				t.Errorf("FileTypeToMimeType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetermineFileType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		fileName    string
		want        artifactpb.File_Type
	}{
		// MIME type based detection
		{"PDF from MIME", "application/pdf", "document.bin", artifactpb.File_TYPE_PDF},
		{"DOCX from MIME", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "doc", artifactpb.File_TYPE_DOCX},
		{"PNG from MIME", "image/png", "image", artifactpb.File_TYPE_PNG},
		{"JPEG from MIME", "image/jpeg", "image", artifactpb.File_TYPE_JPEG},

		// Extension based detection
		{"PDF from extension", "application/octet-stream", "document.pdf", artifactpb.File_TYPE_PDF},
		{"DOCX from extension", "", "document.docx", artifactpb.File_TYPE_DOCX},
		{"PNG from extension", "", "image.png", artifactpb.File_TYPE_PNG},
		{"JPEG from extension (.jpg)", "", "image.jpg", artifactpb.File_TYPE_JPEG},
		{"JPEG from extension (.jpeg)", "", "image.jpeg", artifactpb.File_TYPE_JPEG},
		{"Markdown from extension", "", "readme.md", artifactpb.File_TYPE_MARKDOWN},

		// Unspecified
		{"Unknown type", "application/octet-stream", "file.xyz", artifactpb.File_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetermineFileType(tt.contentType, tt.fileName); got != tt.want {
				t.Errorf("DetermineFileType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileTypeToMediaType(t *testing.T) {
	tests := []struct {
		name     string
		fileType artifactpb.File_Type
		want     artifactpb.File_FileMediaType
	}{
		{"PDF is document", artifactpb.File_TYPE_PDF, artifactpb.File_FILE_MEDIA_TYPE_DOCUMENT},
		{"DOCX is document", artifactpb.File_TYPE_DOCX, artifactpb.File_FILE_MEDIA_TYPE_DOCUMENT},
		{"TEXT is document", artifactpb.File_TYPE_TEXT, artifactpb.File_FILE_MEDIA_TYPE_DOCUMENT},
		{"PNG is image", artifactpb.File_TYPE_PNG, artifactpb.File_FILE_MEDIA_TYPE_IMAGE},
		{"JPEG is image", artifactpb.File_TYPE_JPEG, artifactpb.File_FILE_MEDIA_TYPE_IMAGE},
		{"MP3 is audio", artifactpb.File_TYPE_MP3, artifactpb.File_FILE_MEDIA_TYPE_AUDIO},
		{"OGG is audio", artifactpb.File_TYPE_OGG, artifactpb.File_FILE_MEDIA_TYPE_AUDIO},
		{"MP4 is video", artifactpb.File_TYPE_MP4, artifactpb.File_FILE_MEDIA_TYPE_VIDEO},
		{"MOV is video", artifactpb.File_TYPE_MOV, artifactpb.File_FILE_MEDIA_TYPE_VIDEO},
		{"UNSPECIFIED", artifactpb.File_TYPE_UNSPECIFIED, artifactpb.File_FILE_MEDIA_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileTypeToMediaType(tt.fileType); got != tt.want {
				t.Errorf("FileTypeToMediaType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFileMediaType(t *testing.T) {
	tests := []struct {
		name     string
		fileType artifactpb.File_Type
		mimeType string
		want     artifactpb.File_FileMediaType
	}{
		{"Known file type", artifactpb.File_TYPE_PDF, "", artifactpb.File_FILE_MEDIA_TYPE_DOCUMENT},
		{"Fallback to MIME - image", artifactpb.File_TYPE_UNSPECIFIED, "image/custom", artifactpb.File_FILE_MEDIA_TYPE_IMAGE},
		{"Fallback to MIME - audio", artifactpb.File_TYPE_UNSPECIFIED, "audio/custom", artifactpb.File_FILE_MEDIA_TYPE_AUDIO},
		{"Fallback to MIME - video", artifactpb.File_TYPE_UNSPECIFIED, "video/custom", artifactpb.File_FILE_MEDIA_TYPE_VIDEO},
		{"Fallback to MIME - document", artifactpb.File_TYPE_UNSPECIFIED, "text/plain", artifactpb.File_FILE_MEDIA_TYPE_DOCUMENT},
		{"Unknown", artifactpb.File_TYPE_UNSPECIFIED, "application/custom", artifactpb.File_FILE_MEDIA_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetFileMediaType(tt.fileType, tt.mimeType); got != tt.want {
				t.Errorf("GetFileMediaType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatToFileType(t *testing.T) {
	tests := []struct {
		name   string
		format string
		want   artifactpb.File_Type
	}{
		{"pdf", "pdf", artifactpb.File_TYPE_PDF},
		{"PDF uppercase", "PDF", artifactpb.File_TYPE_PDF},
		{".pdf with dot", ".pdf", artifactpb.File_TYPE_PDF},
		{"png", "png", artifactpb.File_TYPE_PNG},
		{"jpg", "jpg", artifactpb.File_TYPE_JPEG},
		{"jpeg", "jpeg", artifactpb.File_TYPE_JPEG},
		{"mp3", "mp3", artifactpb.File_TYPE_MP3},
		{"mp4", "mp4", artifactpb.File_TYPE_MP4},
		{"ogg", "ogg", artifactpb.File_TYPE_OGG},
		{"markdown", "markdown", artifactpb.File_TYPE_MARKDOWN},
		{"md", "md", artifactpb.File_TYPE_MARKDOWN},
		{"unknown", "unknown", artifactpb.File_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatToFileType(tt.format); got != tt.want {
				t.Errorf("FormatToFileType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertFileTypeString(t *testing.T) {
	tests := []struct {
		name   string
		dbType string
		want   artifactpb.File_Type
	}{
		{"TYPE_PDF", "TYPE_PDF", artifactpb.File_TYPE_PDF},
		{"TYPE_PNG", "TYPE_PNG", artifactpb.File_TYPE_PNG},
		{"TYPE_DOCX", "TYPE_DOCX", artifactpb.File_TYPE_DOCX},
		{"Lowercase", "type_pdf", artifactpb.File_TYPE_PDF},
		{"With whitespace", " TYPE_PDF ", artifactpb.File_TYPE_PDF},
		{"Unknown", "TYPE_UNKNOWN", artifactpb.File_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertFileTypeString(tt.dbType); got != tt.want {
				t.Errorf("ConvertFileTypeString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDataURIPrefix(t *testing.T) {
	tests := []struct {
		name     string
		fileType artifactpb.File_Type
		want     string
	}{
		{"PDF", artifactpb.File_TYPE_PDF, "data:application/pdf;base64,"},
		{"PNG", artifactpb.File_TYPE_PNG, "data:image/png;base64,"},
		{"TEXT", artifactpb.File_TYPE_TEXT, "data:text/plain;base64,"},
		{"UNSPECIFIED", artifactpb.File_TYPE_UNSPECIFIED, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDataURIPrefix(tt.fileType); got != tt.want {
				t.Errorf("GetDataURIPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeedsFileTypeConversion(t *testing.T) {
	tests := []struct {
		name             string
		fileType         artifactpb.File_Type
		wantNeedsConv    bool
		wantTargetFormat string
		wantTargetType   artifactpb.File_Type
	}{
		// Images - standard format
		{"PNG no conversion", artifactpb.File_TYPE_PNG, false, "", artifactpb.File_TYPE_UNSPECIFIED},

		// Images - needs conversion
		{"JPEG to PNG", artifactpb.File_TYPE_JPEG, true, "png", artifactpb.File_TYPE_PNG},
		{"GIF to PNG", artifactpb.File_TYPE_GIF, true, "png", artifactpb.File_TYPE_PNG},
		{"WEBP to PNG", artifactpb.File_TYPE_WEBP, true, "png", artifactpb.File_TYPE_PNG},

		// Audio - standard format
		{"OGG no conversion", artifactpb.File_TYPE_OGG, false, "", artifactpb.File_TYPE_UNSPECIFIED},

		// Audio - needs conversion
		{"MP3 to OGG", artifactpb.File_TYPE_MP3, true, "ogg", artifactpb.File_TYPE_OGG},
		{"WAV to OGG", artifactpb.File_TYPE_WAV, true, "ogg", artifactpb.File_TYPE_OGG},

		// Video - standard format
		{"MP4 no conversion", artifactpb.File_TYPE_MP4, false, "", artifactpb.File_TYPE_UNSPECIFIED},

		// Video - needs conversion
		{"MOV to MP4", artifactpb.File_TYPE_MOV, true, "mp4", artifactpb.File_TYPE_MP4},
		{"AVI to MP4", artifactpb.File_TYPE_AVI, true, "mp4", artifactpb.File_TYPE_MP4},

		// Documents - standard format
		{"PDF no conversion", artifactpb.File_TYPE_PDF, false, "", artifactpb.File_TYPE_UNSPECIFIED},

		// Documents - needs conversion
		{"DOCX to PDF", artifactpb.File_TYPE_DOCX, true, "pdf", artifactpb.File_TYPE_PDF},
		{"DOC to PDF", artifactpb.File_TYPE_DOC, true, "pdf", artifactpb.File_TYPE_PDF},
		{"PPTX to PDF", artifactpb.File_TYPE_PPTX, true, "pdf", artifactpb.File_TYPE_PDF},
		{"HTML to PDF", artifactpb.File_TYPE_HTML, true, "pdf", artifactpb.File_TYPE_PDF},

		// Unknown
		{"UNSPECIFIED", artifactpb.File_TYPE_UNSPECIFIED, false, "", artifactpb.File_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNeedsConv, gotTargetFormat, gotTargetType := NeedsFileTypeConversion(tt.fileType)
			if gotNeedsConv != tt.wantNeedsConv {
				t.Errorf("NeedsFileTypeConversion() needsConversion = %v, want %v", gotNeedsConv, tt.wantNeedsConv)
			}
			if gotTargetFormat != tt.wantTargetFormat {
				t.Errorf("NeedsFileTypeConversion() targetFormat = %v, want %v", gotTargetFormat, tt.wantTargetFormat)
			}
			if gotTargetType != tt.wantTargetType {
				t.Errorf("NeedsFileTypeConversion() targetFileType = %v, want %v", gotTargetType, tt.wantTargetType)
			}
		})
	}
}

func TestFileTypeToExtension(t *testing.T) {
	tests := []struct {
		name     string
		fileType artifactpb.File_Type
		want     string
	}{
		{"PDF", artifactpb.File_TYPE_PDF, "pdf"},
		{"DOCX", artifactpb.File_TYPE_DOCX, "docx"},
		{"PNG", artifactpb.File_TYPE_PNG, "png"},
		{"JPEG", artifactpb.File_TYPE_JPEG, "jpg"},
		{"MP3", artifactpb.File_TYPE_MP3, "mp3"},
		{"MP4", artifactpb.File_TYPE_MP4, "mp4"},
		{"MARKDOWN", artifactpb.File_TYPE_MARKDOWN, "md"},
		{"UNSPECIFIED", artifactpb.File_TYPE_UNSPECIFIED, "bin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileTypeToExtension(tt.fileType); got != tt.want {
				t.Errorf("FileTypeToExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}
