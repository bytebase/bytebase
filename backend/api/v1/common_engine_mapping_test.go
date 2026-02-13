package v1

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestConvertToEngine_AllStoreEnginesMapped(t *testing.T) {
	for _, engine := range allStoreEngines() {
		if engine == storepb.Engine_ENGINE_UNSPECIFIED {
			continue
		}

		got := convertToEngine(engine)
		require.NotEqualf(t, v1pb.Engine_ENGINE_UNSPECIFIED, got, "store engine %q is not mapped in convertToEngine", engine.String())
		require.Equalf(t, engine, convertEngine(got), "store engine %q does not round-trip via v1 mapping", engine.String())
	}
}

func TestConvertEngine_AllV1EnginesMapped(t *testing.T) {
	for _, engine := range allV1Engines() {
		if engine == v1pb.Engine_ENGINE_UNSPECIFIED {
			continue
		}

		got := convertEngine(engine)
		require.NotEqualf(t, storepb.Engine_ENGINE_UNSPECIFIED, got, "v1 engine %q is not mapped in convertEngine", engine.String())
		require.Equalf(t, engine, convertToEngine(got), "v1 engine %q does not round-trip via store mapping", engine.String())
	}
}

func allStoreEngines() []storepb.Engine {
	values := make([]storepb.Engine, 0, len(storepb.Engine_name))
	for value := range storepb.Engine_name {
		values = append(values, storepb.Engine(value))
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})
	return values
}

func allV1Engines() []v1pb.Engine {
	values := make([]v1pb.Engine, 0, len(v1pb.Engine_name))
	for value := range v1pb.Engine_name {
		values = append(values, v1pb.Engine(value))
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})
	return values
}
