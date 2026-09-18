package hierarchy

import (
"errors"
"sync"
)

// Node representa un nodo en la jerarquía
type Node struct {
ID       string   `json:"id"`
ParentID string   `json:"parent_id,omitempty"`
Name     string   `json:"name"`
Type     string   `json:"type"` // org, team, project
Children []string `json:"children"`
Depth    int      `json:"depth"`
Path     string   `json:"path"` // "/root/child/grandchild"
}

// Tree representa la jerarquía completa
type Tree struct {
mu    sync.RWMutex
nodes map[string]*Node
roots []string
}

// NewTree crea un nuevo árbol
func NewTree() *Tree {
return &Tree{nodes: make(map[string]*Node)}
}

// CreateNode crea un nodo hijo
func (t *Tree) CreateNode(parentID, id, name, nodeType string) (*Node, error) {
t.mu.Lock()
defer t.mu.Unlock()

node := &Node{
ID:       id,
ParentID: parentID,
Name:     name,
Type:     nodeType,
Children: []string{},
}

if parentID == "" {
node.Depth = 0
node.Path = "/" + id
t.roots = append(t.roots, id)
} else {
parent, exists := t.nodes[parentID]
if !exists {
return nil, errors.New("parent not found")
}
node.Depth = parent.Depth + 1
node.Path = parent.Path + "/" + id
parent.Children = append(parent.Children, id)
}

t.nodes[id] = node
return node, nil
}

// GetNode devuelve un nodo
func (t *Tree) GetNode(id string) (*Node, bool) {
t.mu.RLock()
defer t.mu.RUnlock()
n, exists := t.nodes[id]
return n, exists
}

// GetChildren devuelve los hijos directos
func (t *Tree) GetChildren(id string) []*Node {
t.mu.RLock()
defer t.mu.RUnlock()

node, exists := t.nodes[id]
if !exists {
return nil
}

result := make([]*Node, 0, len(node.Children))
for _, childID := range node.Children {
if child, ok := t.nodes[childID]; ok {
result = append(result, child)
}
}
return result
}

// GetDescendants devuelve todos los descendientes
func (t *Tree) GetDescendants(id string) []*Node {
t.mu.RLock()
defer t.mu.RUnlock()

var result []*Node
t.collectDescendants(id, &result)
return result
}

func (t *Tree) collectDescendants(id string, result *[]*Node) {
node, exists := t.nodes[id]
if !exists {
return
}
for _, childID := range node.Children {
if child, ok := t.nodes[childID]; ok {
*result = append(*result, child)
t.collectDescendants(childID, result)
}
}
}

// GetAncestors devuelve todos los ancestros
func (t *Tree) GetAncestors(id string) []*Node {
t.mu.RLock()
defer t.mu.RUnlock()

var result []*Node
current, exists := t.nodes[id]
if !exists {
return result
}

for current.ParentID != "" {
parent, ok := t.nodes[current.ParentID]
if !ok {
break
}
result = append([]*Node{parent}, result...)
current = parent
}
return result
}

// Move mueve un nodo a otro parent
func (t *Tree) Move(id, newParentID string) error {
t.mu.Lock()
defer t.mu.Unlock()

node, exists := t.nodes[id]
if !exists {
return errors.New("node not found")
}

// Detach del antiguo parent
if node.ParentID != "" {
if oldParent, ok := t.nodes[node.ParentID]; ok {
oldParent.Children = removeString(oldParent.Children, id)
}
} else {
t.roots = removeString(t.roots, id)
}

// Attach al nuevo parent
if newParentID == "" {
node.ParentID = ""
node.Depth = 0
node.Path = "/" + id
t.roots = append(t.roots, id)
} else {
newParent, ok := t.nodes[newParentID]
if !ok {
return errors.New("new parent not found")
}
node.ParentID = newParentID
node.Depth = newParent.Depth + 1
node.Path = newParent.Path + "/" + id
newParent.Children = append(newParent.Children, id)
}

// Actualizar descendants
t.updatePaths(id)

return nil
}

func (t *Tree) updatePaths(id string) {
node, exists := t.nodes[id]
if !exists {
return
}
for _, childID := range node.Children {
child, ok := t.nodes[childID]
if !ok {
continue
}
child.Depth = node.Depth + 1
child.Path = node.Path + "/" + childID
t.updatePaths(childID)
}
}

func removeString(slice []string, s string) []string {
result := []string{}
for _, item := range slice {
if item != s {
result = append(result, item)
}
}
return result
}

// Size devuelve el número total de nodos
func (t *Tree) Size() int {
t.mu.RLock()
defer t.mu.RUnlock()
return len(t.nodes)
}
