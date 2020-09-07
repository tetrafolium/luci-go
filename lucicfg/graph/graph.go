// Copyright 2018 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package graph

import (
	"container/list"
	"fmt"
	"sort"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/starlark/builtins"
)

var (
	// ErrFinalized is returned by Graph methods that modify the graph when they
	// are used on a finalized graph.
	ErrFinalized = errors.New("cannot modify a finalized graph")

	// ErrNotFinalized is returned by Graph traversal/query methods when they are
	// used with a non-finalized graph.
	ErrNotFinalized = errors.New("cannot query a graph under construction")
)

// backtrace returns an error message annotated with a backtrace.
func backtrace(err error, t *builtins.CapturedStacktrace) string {
	return t.String() + "Error: " + err.Error()
}

// NodeRedeclarationError is returned when a adding an existing node.
type NodeRedeclarationError struct {
	Trace    *builtins.CapturedStacktrace // where it is added second time
	Previous *Node                        // the previously added node
}

// Error is part of 'error' interface.
func (e *NodeRedeclarationError) Error() string {
	// TODO(vadimsh): Improve error messages.
	return fmt.Sprintf("%s is redeclared, previous declaration:\n%s", e.Previous, e.Previous.Trace)
}

// Backtrace returns an error message with a backtrace where it happened.
func (e *NodeRedeclarationError) Backtrace() string {
	return backtrace(e, e.Trace)
}

// CycleError is returned when adding an edge that introduces a cycle.
//
// Nodes referenced by edges may not be fully declared yet (i.e. they may not
// yet have properties or trace associated with them). They do have valid keys
// though.
type CycleError struct {
	Trace *builtins.CapturedStacktrace // where the edge is being added
	Edge  *Edge                        // an edge being added
	Path  []*Edge                      // the rest of the cycle
}

// Error is part of 'error' interface.
func (e *CycleError) Error() string {
	// TODO(vadimsh): Improve error messages.
	return fmt.Sprintf("relation %q between %s and %s introduces a cycle",
		e.Edge.Title, e.Edge.Parent, e.Edge.Child)
}

// Backtrace returns an error message with a backtrace where it happened.
func (e *CycleError) Backtrace() string {
	return backtrace(e, e.Trace)
}

// DanglingEdgeError is returned by Finalize if a graph has an edge whose parent
// or child (or both) haven't been declared by AddNode.
//
// Use Edge.(Parent|Child).Declared() to figure out which end is not defined.
type DanglingEdgeError struct {
	Edge *Edge
}

// Error is part of 'error' interface.
func (e *DanglingEdgeError) Error() string {
	rel := ""
	if e.Edge.Title != "" {
		rel = fmt.Sprintf(" in %q", e.Edge.Title)
	}

	// TODO(vadimsh): Improve error messages.
	hasP := e.Edge.Parent.Declared()
	hasC := e.Edge.Child.Declared()
	switch {
	case hasP && hasC:
		// This should not happen.
		return "incorrect DanglingEdgeError, the edge is fully connected"
	case !hasP && hasC:
		return fmt.Sprintf("%s%s refers to undefined %s",
			e.Edge.Child, rel, e.Edge.Parent)
	case hasP && !hasC:
		return fmt.Sprintf("%s%s refers to undefined %s",
			e.Edge.Parent, rel, e.Edge.Child)
	default:
		return fmt.Sprintf("relation %q: refers to %s and %s, neither is defined",
			e.Edge.Title, e.Edge.Parent, e.Edge.Child)
	}
}

// Backtrace returns an error message with a backtrace where it happened.
func (e *DanglingEdgeError) Backtrace() string {
	return backtrace(e, e.Edge.Trace)
}

// Graph is a DAG of keyed nodes.
//
// It is initially in "under construction" state, in which callers can use
// AddNode and AddEdge (in any order) to build the graph, but can't yet query
// it.
//
// Once the construction is complete, the graph should be finalized via
// Finalize() call, which checks there are no dangling edges and freezes the
// graph (so AddNode/AddEdge return errors), making it queryable.
//
// Graph implements starlark.HasAttrs interface and have the following methods:
//   * key(kind1: string, id1: string, kind2: string, id2: string, ...): graph.key
//   * add_node(key: graph.key, props={}, idempotent=False, trace=stacktrace()): graph.node
//   * add_edge(parent: graph.Key, child: graph.Key, title='', trace=stacktrace())
//   * finalize(): []string
//   * node(key: graph.key): graph.node
//   * children(parent: graph.key, order_by='key'): []graph.node
//   * descendants(root: graph.key, callback=None, order_by='key', topology='breadth'): []graph.node
//   * parents(child: graph.key, order_by='key'): []graph.node
//   * sorted_nodes(nodes: iterable<graph.node>, order_by='key'): []graph.node
type Graph struct {
	KeySet

	nodes     map[*Key]*Node // all declared and "predeclared" nodes
	edges     []*Edge        // all defined edges, in their definition order
	nextIndex int            // index to assign to the next node, for ordering
	finalized bool           // true if the graph is no longer mutable
}

// Visitor visits a node, looks at next possible candidates for a visit and
// returns ones that it really wants to visit.
type Visitor func(n *Node, next []*Node) ([]*Node, error)

// validateOrder returns an error if the edge traversal order is unrecognized.
func validateOrder(orderBy string) error {
	switch orderBy {
	case "key", "~key", "def", "~def":
		return nil
	}
	return fmt.Errorf("unknown order %q, expecting one of \"key\", \"~key\", \"def\", \"~def\"", orderBy)
}

// validateTopology returns an error if the traversal topology is unrecognized.
func validateTopology(topology string) error {
	if topology != "breadth" && topology != "depth" {
		return fmt.Errorf("unknown topology %q, expecting either \"breadth\" or \"depth\"", topology)
	}
	return nil
}

// validateKey returns an error if the key is not from this graph.
func (g *Graph) validateKey(title string, k *Key) error {
	if k.set != &g.KeySet {
		return fmt.Errorf("bad %s: %s is from another graph", title, k)
	}
	return nil
}

// initNode either returns an existing *Node, or adds a new one.
//
// The newly added node is in "predeclared" state: it has no Props or Trace.
// A predeclared node is moved to the fully declared state by AddNode, this can
// happen only once for non-idempotent nodes or more than once for idempotent
// nodes, if each time their props dict is exactly same.
//
// Panics if the graph is finalized.
func (g *Graph) initNode(k *Key) *Node {
	if g.finalized {
		panic(ErrFinalized)
	}
	if n := g.nodes[k]; n != nil {
		return n
	}
	if g.nodes == nil {
		g.nodes = make(map[*Key]*Node, 1)
	}
	n := &Node{Key: k}
	g.nodes[k] = n
	return n
}

//// API to mutate the graph before it is finalized.

// AddNode adds a node to the graph.
//
// If such node already exists, either returns an error right away (if
// 'idempotent' is false), or verifies the existing node has also been marked
// as idempotent and has exact same props dict as being passed here.
//
// Trying to use AddNode after the graph has been finalized is an error.
//
// Freezes props.values() as a side effect.
func (g *Graph) AddNode(k *Key, props *starlark.Dict, idempotent bool, trace *builtins.CapturedStacktrace) error {
	if g.finalized {
		return ErrFinalized
	}
	if err := g.validateKey("key", k); err != nil {
		return err
	}

	// Only string keys are allowed in 'props'.
	for _, pk := range props.Keys() {
		if _, ok := pk.(starlark.String); !ok {
			return fmt.Errorf("non-string key %s in 'props'", pk)
		}
	}
	propsStruct := starlarkstruct.FromKeywords(starlark.String("props"), props.Items())

	n := g.initNode(k)
	if !n.Declared() {
		n.declare(g.nextIndex, propsStruct, idempotent, trace)
		g.nextIndex++
		return nil
	}

	// Only idempotent nodes can be redeclared, and only if all declarations
	// are marked as idempotent and they specify exact same props.
	if n.Idempotent && idempotent {
		switch eq, err := starlark.Equal(propsStruct, n.Props); {
		case err != nil:
			return err
		case eq:
			return nil
		}
	}
	return &NodeRedeclarationError{Trace: trace, Previous: n}
}

// AddEdge adds an edge to the graph.
//
// Trying to use AddEdge after the graph has been finalized is an error.
//
// Neither of the nodes have to exist yet: it is OK to declare nodes and edges
// in arbitrary order as long as at the end of the graph construction (when it
// is finalized) the graph is complete.
//
// It is OK to add the same edge (with the same title) more than once. Only
// the trace of the first definition is recorded.
//
// Returns an error if the new edge introduces a cycle.
func (g *Graph) AddEdge(parent, child *Key, title string, trace *builtins.CapturedStacktrace) error {
	if g.finalized {
		return ErrFinalized
	}
	if err := g.validateKey("parent", parent); err != nil {
		return err
	}
	if err := g.validateKey("child", child); err != nil {
		return err
	}

	edge := &Edge{
		Parent: g.initNode(parent),
		Child:  g.initNode(child),
		Title:  title,
		Trace:  trace,
	}

	// Have this exact edge already? This is fine.
	for _, e := range edge.Parent.children {
		if e.Child == edge.Child && e.Title == title {
			return nil
		}
	}

	// The child may have the parent among its descendants already? Then adding
	// the edge would introduce a cycle.
	err := edge.Child.visitDescendants(nil, func(n *Node, path []*Edge) error {
		if n == edge.Parent {
			return &CycleError{
				Trace: trace,
				Edge:  edge,
				Path:  append([]*Edge(nil), path...),
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	edge.Parent.children = append(edge.Parent.children, edge)
	edge.Child.parents = append(edge.Child.parents, edge)
	g.edges = append(g.edges, edge)
	return nil
}

// Finalize finishes the graph construction by verifying there are no "dangling"
// edges: all edges ever added by AddEdge should connect nodes that were at some
// point defined by AddNode.
//
// Finalizing an already finalized graph is not an error.
//
// A finalized graph is immutable (and frozen in Starlark sense): all subsequent
// calls to AddNode/AddEdge return an error. Conversely, freezing the graph via
// Freeze() finalizes it, panicking if the finalization fails. Users of Graph
// are expected to finalize the graph themselves (checking errors) before
// Starlark tries to freeze it.
//
// Once finalized, the graph becomes queryable.
func (g *Graph) Finalize() (errs errors.MultiError) {
	if g.finalized {
		return
	}

	for _, e := range g.edges {
		if !e.Parent.Declared() || !e.Child.Declared() {
			errs = append(errs, &DanglingEdgeError{Edge: e})
		}
	}

	g.finalized = len(errs) == 0
	return
}

//// API to query the graph after it is finalized.

// Node returns a node by the key or (nil, nil) if there's no such node.
//
// Trying to use Node before the graph has been finalized is an error.
func (g *Graph) Node(k *Key) (*Node, error) {
	if !g.finalized {
		return nil, ErrNotFinalized
	}
	if err := g.validateKey("key", k); err != nil {
		return nil, err
	}
	switch n := g.nodes[k]; {
	case n == nil:
		return nil, nil
	case !n.Declared():
		panic(fmt.Errorf("impossible not-yet-declared node in a finalized graph: %s", n.Key))
	default:
		return n, nil
	}
}

// Children returns direct children of a node (given by its key).
//
// The order of the result depends on a value of 'orderBy':
//   'key': nodes are ordered lexicographically by their keys (smaller first).
//   'def': nodes are ordered by the order edges to them were defined during
//        the execution (earlier first).
//
// Any other value of 'orderBy' causes an error.
//
// A missing node is considered to have no children.
//
// Trying to use Children before the graph has been finalized is an error.
func (g *Graph) Children(parent *Key, orderBy string) ([]*Node, error) {
	return g.orderedRelatives(parent, "parent", orderBy, (*Node).listChildren)
}

// Descendants recursively visits 'root' and all its children, in breadth or
// depth first order.
//
// Returns the list of all visited nodes, in order they were visited, including
// 'root' itself. If 'root' is missing, returns an empty list.
//
// The order of enumeration of direct children of a node depends on a value of
// 'orderBy':
//   'key': nodes are ordered lexicographically by their keys (smaller first).
//   'def': nodes are ordered by the order edges to them were defined during
//        the execution (earlier first).
//
// '~' in front (i.e. ~key' and '~def') means "reverse order".
//
// Any other value of 'orderBy' causes an error.
//
// Each node is visited only once, even if it is reachable through multiple
// paths. Note that the graph has no cycles (by construction).
//
// The visitor callback (if not nil) is called for each visited node. It decides
// what children to visit next. The callback always sees all children of the
// node, even if some of them (or all) have already been visited. Visited nodes
// will be skipped even if the visitor returns them.
//
// Trying to use Descendants before the graph has been finalized is an error.
func (g *Graph) Descendants(root *Key, orderBy, topology string, visitor Visitor) ([]*Node, error) {
	if !g.finalized {
		return nil, ErrNotFinalized
	}
	if err := g.validateKey("root", root); err != nil {
		return nil, err
	}
	if err := validateOrder(orderBy); err != nil {
		return nil, err
	}
	if err := validateTopology(topology); err != nil {
		return nil, err
	}

	rootNode := g.nodes[root]
	if rootNode == nil {
		return nil, nil
	}

	switch topology {
	case "breadth":
		return descBreadth(rootNode, orderBy, visitor)
	case "depth":
		return descDepth(rootNode, orderBy, visitor)
	default:
		panic("impossible")
	}
}

// descBreadth implements non-recursive breadth-first traversal.
func descBreadth(root *Node, orderBy string, visitor Visitor) ([]*Node, error) {
	queue := list.New()
	queuedSet := make(map[*Node]struct{}, 1)
	visited := make([]*Node, 0, 1)

	enqueue := func(n *Node) {
		if _, yes := queuedSet[n]; !yes {
			queue.PushBack(n)
			queuedSet[n] = struct{}{}
		}
	}
	enqueue(root)

	for queue.Len() > 0 {
		cur := queue.Remove(queue.Front()).(*Node)
		visited = append(visited, cur)

		// Ask the visitor callback to decide where it wants to go next.
		next, err := filteredChildren(cur, orderBy, visitor)
		if err != nil {
			return nil, err
		}

		// Go there (eventually).
		for _, n := range next {
			enqueue(n)
		}
	}
	return visited, nil
}

// descDepth implements recursive depth-first traversal.
func descDepth(root *Node, orderBy string, visitor Visitor) ([]*Node, error) {
	visited := make([]*Node, 0, 1)
	visitedSet := make(map[*Node]struct{}, 1)

	var visit func(root *Node) error
	visit = func(root *Node) error {
		// Been here before?
		if _, yes := visitedSet[root]; yes {
			return nil
		}
		visitedSet[root] = struct{}{}

		// Ask the visitor callback to decide where it wants to go next.
		next, err := filteredChildren(root, orderBy, visitor)
		if err != nil {
			return err
		}

		// Go there, right now.
		for _, n := range next {
			if err := visit(n); err != nil {
				return err
			}
		}

		// Now that all children are visited, emit the root itself too.
		visited = append(visited, root)
		return nil
	}

	if err := visit(root); err != nil {
		return nil, err
	}
	return visited, nil
}

// filteredChildren calls 'visitor' callback to filter the list of children.
//
// Used by Descendants implementation.
func filteredChildren(cur *Node, orderBy string, visitor Visitor) ([]*Node, error) {
	children := sortByEdgeOrder(cur.listChildren(), orderBy)
	if visitor == nil {
		return children, nil
	}
	// Ask the visitor callback to decide where it wants to go next. Verify the
	// callback didn't sneakily return some *Node it saved somewhere before.
	// This may cause infinite recursion and other weird stuff.
	next, err := visitor(cur, children)
	if err != nil {
		return nil, err
	}
	allowed := make(map[*Node]struct{}, len(children))
	for _, n := range children {
		allowed[n] = struct{}{}
	}
	for _, n := range next {
		if _, yes := allowed[n]; !yes {
			return nil, fmt.Errorf("the callback unexpectedly returned %s which is not a child of %s", n, cur)
		}
	}
	return next, nil
}

// Parents returns direct parents of a node (given by its key).
//
// The order of the result depends on a value of 'orderBy':
//   'key': nodes are ordered lexicographically by their keys (smaller first).
//   'def': nodes are ordered by the order edges to them were defined during
//        the execution (earlier first).
//
// Any other value of 'orderBy' causes an error.
//
// A missing node is considered to have no parents.
//
// Trying to use Parents before the graph has been finalized is an error.
func (g *Graph) Parents(child *Key, orderBy string) ([]*Node, error) {
	return g.orderedRelatives(child, "child", orderBy, (*Node).listParents)
}

// SortNodes sorts a slice of nodes of this graph in-place.
//
// The order of the result depends on the value of 'orderBy':
//   'key': nodes are ordered lexicographically by their keys (smaller first).
//   'def': nodes are ordered by the order they were defined in the graph.
//
// Any other value of 'orderBy' causes an error.
func (g *Graph) SortNodes(nodes []*Node, orderBy string) error {
	if err := validateOrder(orderBy); err != nil {
		return err
	}

	// Only nodes from the same graph are comparable to each other. Comparing
	// .Index from different graphs makes no sense.
	for _, n := range nodes {
		if !n.BelongsTo(g) {
			return fmt.Errorf("bad node %s - from another graph", n)
		}
	}

	switch orderBy {
	case "def":
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].Index < nodes[j].Index })
	case "key":
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].Key.Less(nodes[j].Key) })
	default:
		// Must not happen, orderBy must already be validated here.
		panic(fmt.Sprintf("unknown order %q", orderBy))
	}
	return nil
}

// orderedRelatives is a common implementation of Children and Parents.
func (g *Graph) orderedRelatives(key *Key, attr, orderBy string, cb func(n *Node) []*Node) ([]*Node, error) {
	if !g.finalized {
		return nil, ErrNotFinalized
	}
	if err := g.validateKey(attr, key); err != nil {
		return nil, err
	}
	if err := validateOrder(orderBy); err != nil {
		return nil, err
	}
	if n := g.nodes[key]; n != nil {
		return sortByEdgeOrder(cb(n), orderBy), nil
	}
	return nil, nil // no node at all -> no related nodes
}

// sortByEdgeOrder orders either children or parents of some node according to
// the order of edges to/from them.
//
// Assumes 'nodes' came from either .listChildren() or .listParents(), and thus
// are mutable copies, which are already ordered by 'def' order.
//
// Note that 'def' order here means "in order edges were defined". This is
// different from "in order nodes themselves were defined". This is different
// from the meaning of 'def' order in SortNodes.
func sortByEdgeOrder(nodes []*Node, orderBy string) []*Node {
	switch orderBy {
	case "def":
		// default
	case "~def":
		for i, j := 0, len(nodes)-1; i < j; i, j = i+1, j-1 {
			nodes[i], nodes[j] = nodes[j], nodes[i]
		}
	case "key":
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].Key.Less(nodes[j].Key) })
	case "~key":
		sort.Slice(nodes, func(i, j int) bool { return nodes[j].Key.Less(nodes[i].Key) })
	default:
		// Must not happen, orderBy must already be validated here.
		panic(fmt.Sprintf("unknown order %q", orderBy))
	}
	return nodes
}

//// starlark.Value interface implementation.

// String is a part of starlark.Value interface
func (g *Graph) String() string { return "graph" }

// Type is a part of starlark.Value interface.
func (g *Graph) Type() string { return "graph" }

// Freeze is a part of starlark.Value interface.
//
// It finalizes the graph, panicking on errors. Users of Graph are expected to
// finalize the graph themselves (checking errors) before Starlark tries to
// freeze it.
func (g *Graph) Freeze() {
	if err := g.Finalize(); err != nil {
		panic(err)
	}
}

// Truth is a part of starlark.Value interface.
func (g *Graph) Truth() starlark.Bool { return starlark.True }

// Hash is a part of starlark.Value interface.
func (g *Graph) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable") }

// AttrNames is a part of starlark.HasAttrs interface.
func (g *Graph) AttrNames() []string {
	names := make([]string, 0, len(graphAttrs))
	for k := range graphAttrs {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// Attr is a part of starlark.HasAttrs interface.
func (g *Graph) Attr(name string) (starlark.Value, error) {
	impl, ok := graphAttrs[name]
	if !ok {
		return nil, nil // per Attr(...) contract
	}
	return impl.BindReceiver(g), nil
}

//// Starlark bindings for individual graph methods.

func nodesList(nodes []*Node) *starlark.List {
	vals := make([]starlark.Value, len(nodes))
	for i, n := range nodes {
		vals[i] = n
	}
	return starlark.NewList(vals)
}

var graphAttrs = map[string]*starlark.Builtin{
	// key(kind1: string, id1: string, kind2: string, id2: string, ...): graph.key
	"key": starlark.NewBuiltin("key", func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if len(kwargs) != 0 {
			return nil, fmt.Errorf("graph.key: not expecting keyword arguments")
		}
		pairs := make([]string, len(args))
		for idx, arg := range args {
			str, ok := arg.(starlark.String)
			if !ok {
				return nil, fmt.Errorf("graph.key: all arguments must be strings, arg #%d was %s", idx, arg.Type())
			}
			pairs[idx] = str.GoString()
		}
		return b.Receiver().(*Graph).Key(pairs...)
	}),

	// add_node(key: graph.key, props={}, idempotent=False, trace=stacktrace()): graph.node
	"add_node": starlark.NewBuiltin("add_node", func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var key *Key
		var props *starlark.Dict
		var idempotent starlark.Bool
		var trace *builtins.CapturedStacktrace
		err := starlark.UnpackArgs("add_node", args, kwargs,
			"key", &key,
			"props?", &props,
			"idempotent?", &idempotent,
			"trace?", &trace,
		)
		if err != nil {
			return nil, err
		}
		if props == nil {
			props = &starlark.Dict{}
		}
		if trace == nil {
			var err error
			if trace, err = builtins.CaptureStacktrace(th, 0); err != nil {
				return nil, err
			}
		}
		err = b.Receiver().(*Graph).AddNode(key, props, bool(idempotent), trace)
		return starlark.None, err
	}),

	// add_edge(parent: graph.Key, child: graph.Key, title='', trace=stacktrace())
	"add_edge": starlark.NewBuiltin("add_edge", func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var parent *Key
		var child *Key
		var title starlark.String
		var trace *builtins.CapturedStacktrace
		err := starlark.UnpackArgs("add_edge", args, kwargs,
			"parent", &parent,
			"child", &child,
			"title?", &title,
			"trace?", &trace)
		if err != nil {
			return nil, err
		}

		if trace == nil {
			var err error
			if trace, err = builtins.CaptureStacktrace(th, 0); err != nil {
				return nil, err
			}
		}

		err = b.Receiver().(*Graph).AddEdge(parent, child, title.GoString(), trace)
		return starlark.None, err
	}),

	// finalize(): []string
	"finalize": starlark.NewBuiltin("finalize", func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := starlark.UnpackArgs("finalize", args, kwargs); err != nil {
			return nil, err
		}
		errs := b.Receiver().(*Graph).Finalize()
		out := make([]starlark.Value, len(errs))
		for i, err := range errs {
			out[i] = starlark.String(err.Error())
		}
		return starlark.NewList(out), nil
	}),

	// node(key: graph.key): graph.node
	"node": starlark.NewBuiltin("node", func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var key *Key
		if err := starlark.UnpackArgs("node", args, kwargs, "key", &key); err != nil {
			return nil, err
		}
		if node, err := b.Receiver().(*Graph).Node(key); node != nil || err != nil {
			return node, err
		}
		return starlark.None, nil
	}),

	// children(parent: graph.key, order_by='key'): []graph.node
	"children": starlark.NewBuiltin("children", func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var parent *Key
		var orderBy starlark.String = "key"
		err := starlark.UnpackArgs("children", args, kwargs,
			"parent", &parent,
			"order_by?", &orderBy)
		if err != nil {
			return nil, err
		}
		nodes, err := b.Receiver().(*Graph).Children(parent, orderBy.GoString())
		if err != nil {
			return nil, err
		}
		return nodesList(nodes), nil
	}),

	// descendants(root: graph.key, callback=None, order_by='key'): []graph.Node.
	//
	// where 'callback' is func(node graph.node, children []graph.node): []graph.node.
	"descendants": starlark.NewBuiltin("descendants", func(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var root *Key
		var callback starlark.Value
		var orderBy starlark.String = "key"
		var topology starlark.String = "breadth"
		err := starlark.UnpackArgs("descendants", args, kwargs,
			"root", &root,
			"callback?", &callback,
			"order_by?", &orderBy,
			"topology?", &topology)
		if err != nil {
			return nil, err
		}

		// Glue layer between Go callback and Starlark callback.
		var visitor Visitor
		if callback != nil && callback != starlark.None {
			visitor = func(node *Node, children []*Node) ([]*Node, error) {
				// Call callback(node, children).
				ret, err := starlark.Call(th, callback, starlark.Tuple{node, nodesList(children)}, nil)
				if err != nil {
					return nil, err
				}
				// The callback is expected to return a list.
				lst, _ := ret.(*starlark.List)
				if lst == nil {
					return nil, fmt.Errorf(
						"descendants: callback %s unexpectedly returned %s instead of a list",
						callback, ret.Type())
				}
				// And it should be a list of nodes.
				out := make([]*Node, lst.Len())
				for i := range out {
					if out[i], _ = lst.Index(i).(*Node); out[i] == nil {
						return nil, fmt.Errorf(
							"descendants: callback %s unexpectedly returned %s as element #%d instead of a graph.node",
							callback, lst.Index(i).Type(), i)
					}
				}
				return out, nil
			}
		}

		nodes, err := b.Receiver().(*Graph).Descendants(root, orderBy.GoString(), topology.GoString(), visitor)
		if err != nil {
			return nil, err
		}
		return nodesList(nodes), nil
	}),

	// parents(child: graph.key, order_by='key'): []graph.node
	"parents": starlark.NewBuiltin("parents", func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var child *Key
		var orderBy starlark.String = "key"
		err := starlark.UnpackArgs("parents", args, kwargs,
			"child", &child,
			"order_by?", &orderBy)
		if err != nil {
			return nil, err
		}
		nodes, err := b.Receiver().(*Graph).Parents(child, orderBy.GoString())
		if err != nil {
			return nil, err
		}
		return nodesList(nodes), nil
	}),

	// sorted_nodes(nodes: iterable<graph.node>, order_by='key'): []graph.node
	"sorted_nodes": starlark.NewBuiltin("sorted_nodes", func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var nodes starlark.Iterable
		var orderBy starlark.String = "key"
		err := starlark.UnpackArgs("sorted_nodes", args, kwargs,
			"nodes", &nodes,
			"order_by?", &orderBy)
		if err != nil {
			return nil, err
		}

		iter := nodes.Iterate()
		defer iter.Done()

		var toSort []*Node
		var val starlark.Value
		for iter.Next(&val) {
			node, ok := val.(*Node)
			if !ok {
				return nil, fmt.Errorf("sorted_nodes: got %s, expecting graph.node", val)
			}
			toSort = append(toSort, node)
		}

		if err := b.Receiver().(*Graph).SortNodes(toSort, orderBy.GoString()); err != nil {
			return nil, err
		}
		return nodesList(toSort), nil
	}),
}
