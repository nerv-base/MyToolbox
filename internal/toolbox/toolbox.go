// Package toolbox is the registration boundary for built-in native tools.
// A tool owns its view and state. The shell knows only its metadata and lifetime.
package toolbox

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

type Info struct {
	ID, Name, Description string
	Icon                  *ui.SVG
}

type Tool interface {
	Info() Info
	View(*ui.Context)
	Close()
}

// FileReceiver is optional: tools that do not process files need not implement it.
type FileReceiver interface{ AddFiles([]string) }

type Host struct {
	// Update marshals every background state change onto the UI thread.
	Update     func(func())
	PickFiles  func(title string, extensions []string, done func([]string, error))
	OpenFile   func(string)
	RevealFile func(string)
}

type Registry struct {
	tools []Tool
	byID  map[string]Tool
}

func (r *Registry) Register(tool Tool) error {
	info := tool.Info()
	if info.ID == "" || info.Name == "" {
		return fmt.Errorf("tool ID and name are required")
	}
	if r.byID == nil {
		r.byID = make(map[string]Tool)
	}
	if _, exists := r.byID[info.ID]; exists {
		return fmt.Errorf("duplicate tool ID: %s", info.ID)
	}
	r.byID[info.ID] = tool
	r.tools = append(r.tools, tool)
	return nil
}

func (r *Registry) Tools() []Tool      { return append([]Tool(nil), r.tools...) }
func (r *Registry) Get(id string) Tool { return r.byID[id] }
func (r *Registry) Close() {
	for _, tool := range r.tools {
		tool.Close()
	}
}
