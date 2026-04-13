package file

import (
	"testing"

	artifactpb "github.com/instill-ai/protogen-go/artifact/v1alpha"
)

func TestNeedFileTypeConversion(t *testing.T) {
	tests := []struct {
		name           string
		fileType       artifactpb.File_Type
		wantNeed       bool
		wantFormat     string
		wantTargetType artifactpb.File_Type
	}{
		{
			name:           "XLSX needs no conversion",
			fileType:       artifactpb.File_TYPE_XLSX,
			wantNeed:       false,
			wantFormat:     "",
			wantTargetType: artifactpb.File_TYPE_UNSPECIFIED,
		},
		{
			name:           "XLS converts to XLSX",
			fileType:       artifactpb.File_TYPE_XLS,
			wantNeed:       true,
			wantFormat:     "xlsx",
			wantTargetType: artifactpb.File_TYPE_XLSX,
		},
		{
			name:           "PDF needs no conversion",
			fileType:       artifactpb.File_TYPE_PDF,
			wantNeed:       false,
			wantFormat:     "",
			wantTargetType: artifactpb.File_TYPE_UNSPECIFIED,
		},
		{
			name:           "DOCX converts to PDF",
			fileType:       artifactpb.File_TYPE_DOCX,
			wantNeed:       true,
			wantFormat:     "pdf",
			wantTargetType: artifactpb.File_TYPE_PDF,
		},
		{
			name:           "DOC converts to PDF",
			fileType:       artifactpb.File_TYPE_DOC,
			wantNeed:       true,
			wantFormat:     "pdf",
			wantTargetType: artifactpb.File_TYPE_PDF,
		},
		{
			name:           "PNG needs no conversion",
			fileType:       artifactpb.File_TYPE_PNG,
			wantNeed:       false,
			wantFormat:     "",
			wantTargetType: artifactpb.File_TYPE_UNSPECIFIED,
		},
		{
			name:           "GIF converts to PNG",
			fileType:       artifactpb.File_TYPE_GIF,
			wantNeed:       true,
			wantFormat:     "png",
			wantTargetType: artifactpb.File_TYPE_PNG,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNeed, gotFormat, gotTarget := NeedFileTypeConversion(tt.fileType)
			if gotNeed != tt.wantNeed {
				t.Errorf("NeedFileTypeConversion(%v).need = %v, want %v", tt.fileType, gotNeed, tt.wantNeed)
			}
			if gotFormat != tt.wantFormat {
				t.Errorf("NeedFileTypeConversion(%v).format = %q, want %q", tt.fileType, gotFormat, tt.wantFormat)
			}
			if gotTarget != tt.wantTargetType {
				t.Errorf("NeedFileTypeConversion(%v).targetType = %v, want %v", tt.fileType, gotTarget, tt.wantTargetType)
			}
		})
	}
}

func TestGetConvertedFileTypeInfo_Spreadsheets(t *testing.T) {
	xlsxMIME := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

	tests := []struct {
		name     string
		fileType artifactpb.File_Type
		wantExt  string
		wantMIME string
	}{
		{"XLSX standardizes to xlsx", artifactpb.File_TYPE_XLSX, "xlsx", xlsxMIME},
		{"XLS standardizes to xlsx", artifactpb.File_TYPE_XLS, "xlsx", xlsxMIME},
		{"PDF standardizes to pdf", artifactpb.File_TYPE_PDF, "pdf", "application/pdf"},
		{"DOCX standardizes to pdf", artifactpb.File_TYPE_DOCX, "pdf", "application/pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gotExt, gotMIME := GetConvertedFileTypeInfo(tt.fileType)
			if gotExt != tt.wantExt {
				t.Errorf("GetConvertedFileTypeInfo(%v).ext = %q, want %q", tt.fileType, gotExt, tt.wantExt)
			}
			if gotMIME != tt.wantMIME {
				t.Errorf("GetConvertedFileTypeInfo(%v).mime = %q, want %q", tt.fileType, gotMIME, tt.wantMIME)
			}
		})
	}
}
