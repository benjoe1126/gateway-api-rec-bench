package patch

type JsonPatchOp string

const (
	PatchOpAdd     JsonPatchOp = "add"
	PatchOpRemove  JsonPatchOp = "remove"
	PatchOpReplace JsonPatchOp = "replace"
	PatchOpMove    JsonPatchOp = "move"
	PatchOpCopy    JsonPatchOp = "copy"
	PatchOpTest    JsonPatchOp = "test"
)

type JsonPatch struct {
	Op    JsonPatchOp `json:"op" yaml:"op"`
	From  string      `json:"from,omitempty" yaml:"from,omitempty"`
	Path  string      `json:"path" yaml:"path"`
	Value interface{} `json:"value,omitempty" yaml:"value,omitempty"`
}
