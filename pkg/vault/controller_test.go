package vault

import (
	"reflect"
	"testing"
)

func Test_mergeAnnotations(t *testing.T) {
	tests := []struct {
		name            string
		userAnnotations map[string]string
		baseAnnotations map[string]string
		want            map[string]string
	}{
		{
			name:            "merge annotations maps",
			userAnnotations: map[string]string{"hello": "world"},
			baseAnnotations: map[string]string{"foo": "bar"},
			want:            map[string]string{"hello": "world", "foo": "bar"},
		},
		{
			name:            "user annotations will not overwrite base annotations",
			userAnnotations: map[string]string{"hello": "universe"},
			baseAnnotations: map[string]string{"hello": "world"},
			want:            map[string]string{"hello": "world"},
		},
		{
			name:            "user annotations is empty",
			userAnnotations: map[string]string{},
			baseAnnotations: map[string]string{"hello": "world"},
			want:            map[string]string{"hello": "world"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeAnnotations(tt.userAnnotations, tt.baseAnnotations); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeAnnotations() = %v, want %v", got, tt.want)
			}
		})
	}
}
