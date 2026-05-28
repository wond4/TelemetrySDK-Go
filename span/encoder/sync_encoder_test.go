package encoder

import (
	"github.com/wond4/TelemetrySDK-Go/span/v2/exporter"
	"reflect"
	"testing"
)

func TestNewSyncEncoder(t *testing.T) {
	type args struct {
		e exporter.SyncExporter
	}
	tests := []struct {
		name string
		args args
		want SyncEncoder
	}{
		{
			"TestNewSyncEncoder",
			args{exporter.SyncRealTimeExporter()},
			&SyncJsonEncoder{syncExporter: exporter.SyncRealTimeExporter()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSyncEncoder(tt.args.e); !reflect.DeepEqual(got.Close(), tt.want.Close()) {
				t.Errorf("NewSyncEncoder(%v), want %v", got, tt.want)
			}
		})
	}
}
