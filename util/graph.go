package util

import (
	"container/list"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Entity[T any] struct {
	Lock *sync.RWMutex

	Id       string     `json:"id" bson:"_id"`
	Type     string     `json:"type" bson:"type"`
	Data     T          `json:"data,omitempty" bson:"data,omitempty"`
	Created  time.Time  `json:"created" bson:"created"`
	Modified *time.Time `json:"modified,omitempty" bson:"modified,omitempty"`
}

type Edge[T any] struct {
	Entity[T]

	SourceId string `json:"source_ref" bson:"source_ref"`
	TargetId string `json:"target_ref" bson:"target_ref"`
}

type Node[T any] struct {
	Entity[T]

	OutgoingEdges Stream[string]
	IngoingEdges  Stream[string]
}

type Graph[N, E any] struct {
	Entity[*Graph[N, E]]

	Created  time.Time  `json:"created" bson:"created"`
	Modified *time.Time `json:"modified,omitempty" bson:"modified,omitempty"`

	Edges            Stream[*Edge[E]] `json:"edges,omitempty" bson:"edges,omitempty"`
	Nodes            Stream[*Node[N]] `json:"nodes,omitempty" bson:"nodes,omitempty"`
	ExistingEntities Map[string, any] `json:"-" bson:"-"`

	EdgeKeyGen func(from, to string, data E) string
	NodeKeyGen func(n N) string
}

func NewGraph[N, E any](nodeIdGen func(N) any, edgeIdGen ...func(string, string, E) any) *Graph[N, E] {
	if nodeIdGen == nil {
		nodeIdGen = func(n N) any {
			h, _ := Hash(n)
			return h
		}
	}

	if len(edgeIdGen) == 0 {
		n := time.Now()
		edgeIdGen = []func(string, string, E) any{
			func(from string, to string, _ E) any {
				return fmt.Sprint(from, "-->", n.UnixNano(), "-->", to)
			},
		}
	}

	return &Graph[N, E]{
		Entity: Entity[*Graph[N, E]]{
			Lock:     &sync.RWMutex{},
			Id:       GenerateUUID("graph", time.Now()),
			Type:     "graph",
			Data:     nil,
			Created:  time.Now(),
			Modified: nil,
		},
		ExistingEntities: Map[string, any]{},
		NodeKeyGen: func(n N) string {
			return GenerateUUID("node", nodeIdGen(n))
		},
		EdgeKeyGen: func(from, to string, data E) string {
			return GenerateUUID("edge", edgeIdGen[0](from, to, data))
		},
	}
}

func (g *Graph[N, E]) AddNode(data N) string {
	id := g.NodeKeyGen(data)

	g.Lock.Lock()
	if node, exist := g.ExistingEntities[id].(*Node[N]); exist {
		node.Data = data
		n := time.Now()
		node.Modified = &n
	} else {
		n := &Node[N]{
			Entity: Entity[N]{
				Id:      id,
				Data:    data,
				Type:    "node",
				Created: time.Now(),
				Lock:    &sync.RWMutex{},
			},
		}

		g.Nodes = append(g.Nodes, n)
		g.ExistingEntities[id] = n
	}

	g.Lock.Unlock()

	return id
}

func (g *Graph[N, E]) AddEdge(from, to string, data E) string {
	id := g.EdgeKeyGen(from, to, data)

	g.Lock.Lock()
	if edge, exist := g.ExistingEntities[id].(*Edge[E]); exist {
		edge.Data = data
		n := time.Now()
		edge.Modified = &n
	} else {
		e := &Edge[E]{
			Entity: Entity[E]{
				Lock:    &sync.RWMutex{},
				Id:      id,
				Data:    data,
				Type:    "edge",
				Created: time.Now(),
			},
			SourceId: from,
			TargetId: to,
		}

		g.Edges = append(g.Edges, e)
		g.ExistingEntities[id] = e

		if node, exist := g.ExistingEntities[from].(*Node[E]); exist {
			node.IngoingEdges = append(node.IngoingEdges, from)
		}

		if node, exist := g.ExistingEntities[to].(*Node[E]); exist {
			node.OutgoingEdges = append(node.OutgoingEdges, to)
		}
	}

	g.Lock.Unlock()

	return id
}

func (g *Graph[N, E]) Entities() Stream[any] {
	res := make([]any, len(g.Nodes)+len(g.Edges))

	var index int

	g.Lock.RLock()
	for _, n := range g.Nodes {
		res[index] = *n
		index++
	}

	for _, e := range g.Edges {
		res[index] = *e
		index++
	}

	g.Lock.RUnlock()

	return res
}

func (g *Graph[N, _]) GetNode(data N) *Node[N] {
	return g.GetNodeById(g.NodeKeyGen(data))
}

func (g *Graph[N, _]) GetNodeById(id string) *Node[N] {
	g.Lock.RLock()
	n := g.ExistingEntities.Get(id).(*Node[N])
	g.Lock.RUnlock()

	return n
}

func (g *Graph[_, E]) GetEdge(from, to string, data E) *Edge[E] {
	return g.GetEdgeById(g.EdgeKeyGen(from, to, data))
}

func (g *Graph[_, E]) GetEdgeById(id string) *Edge[E] {
	g.Lock.RLock()
	e := g.ExistingEntities.Get(id).(*Edge[E])
	g.Lock.RUnlock()

	return e
}

func (g *Graph[N, E]) ForEachEdge(consumer func(edge *Edge[E])) {
	g.Lock.RLock()
	g.Edges.ForEach(consumer)
	g.Lock.RUnlock()
}

func (g *Graph[N, E]) ForEachNode(consumer func(node *Node[N])) {
	g.Lock.RLock()
	g.Nodes.ForEach(consumer)
	g.Lock.RUnlock()
}

func (g *Graph[N, E]) BFS(startNodeId string) Stream[string] {
	if startNodeId == "" {
		return []string{}
	}

	g.Lock.RLock()
	defer g.Lock.RUnlock()

	if _, exists := g.ExistingEntities[startNodeId]; !exists {
		return []string{}
	}

	visited := make(map[string]bool, g.Nodes.Len())
	result := make(Stream[string], 0, g.Nodes.Len())

	queue := make(Stream[string], 0, g.Nodes.Len())

	queue = append(queue, startNodeId)
	visited[startNodeId] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		result = append(result, current)
		if nodeEntity, exists := g.ExistingEntities[current]; exists {
			if node, ok := nodeEntity.(*Node[N]); ok {
				for _, edgeId := range node.OutgoingEdges {
					if edgeEntity, edgeExists := g.ExistingEntities[edgeId]; edgeExists {
						if edge, edgeOk := edgeEntity.(*Edge[E]); edgeOk {
							targetId := edge.TargetId
							if !visited[targetId] {
								visited[targetId] = true
								queue = append(queue, targetId)
							}
						}
					}
				}
			}
		}
	}

	return result
}

func (g *Graph[N, E]) BFSWithCallback(startNodeId string, callback func(node *Node[N]) bool) {
	if startNodeId == "" || callback == nil {
		return
	}

	g.Lock.RLock()
	defer g.Lock.RUnlock()

	if _, exists := g.ExistingEntities[startNodeId]; !exists {
		return
	}

	visited := make(map[string]bool, len(g.Nodes))
	queue := make([]string, 0, len(g.Nodes))

	queue = append(queue, startNodeId)
	visited[startNodeId] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if nodeEntity, exists := g.ExistingEntities[current]; exists {
			if node, ok := nodeEntity.(*Node[N]); ok {
				if !callback(node) {
					return
				}

				for _, edgeId := range node.OutgoingEdges {
					if edgeEntity, edgeExists := g.ExistingEntities[edgeId]; edgeExists {
						if edge, edgeOk := edgeEntity.(*Edge[E]); edgeOk {
							targetId := edge.TargetId
							if !visited[targetId] {
								visited[targetId] = true
								queue = append(queue, targetId)
							}
						}
					}
				}
			}
		}
	}
}

func (g *Graph[N, E]) DFS(startNodeId string) Stream[string] {
	if startNodeId == "" {
		return []string{}
	}

	g.Lock.RLock()
	defer g.Lock.RUnlock()

	if _, exists := g.ExistingEntities[startNodeId]; !exists {
		return []string{}
	}

	visited := make(map[string]bool, len(g.Nodes))
	result := make([]string, 0, len(g.Nodes))

	stack := make([]string, 0, len(g.Nodes))

	stack = append(stack, startNodeId)

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[current] {
			continue
		}

		visited[current] = true
		result = append(result, current)

		if nodeEntity, exists := g.ExistingEntities[current]; exists {
			if node, ok := nodeEntity.(*Node[N]); ok {
				for i := len(node.OutgoingEdges) - 1; i >= 0; i-- {
					edgeId := node.OutgoingEdges[i]
					if edgeEntity, edgeExists := g.ExistingEntities[edgeId]; edgeExists {
						if edge, edgeOk := edgeEntity.(*Edge[E]); edgeOk {
							targetId := edge.TargetId
							if !visited[targetId] {
								stack = append(stack, targetId)
							}
						}
					}
				}
			}
		}
	}

	return result
}

func (g *Graph[N, E]) DFSWithCallback(startNodeId string, callback func(node *Node[N]) bool) {
	if startNodeId == "" || callback == nil {
		return
	}

	g.Lock.RLock()
	defer g.Lock.RUnlock()

	if _, exists := g.ExistingEntities[startNodeId]; !exists {
		return
	}

	visited := make(map[string]bool, len(g.Nodes))
	stack := make([]string, 0, len(g.Nodes))

	stack = append(stack, startNodeId)

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[current] {
			continue
		}

		visited[current] = true

		if nodeEntity, exists := g.ExistingEntities[current]; exists {
			if node, ok := nodeEntity.(*Node[N]); ok {
				if !callback(node) {
					return // Early termination
				}

				for i := len(node.OutgoingEdges) - 1; i >= 0; i-- {
					edgeId := node.OutgoingEdges[i]
					if edgeEntity, edgeExists := g.ExistingEntities[edgeId]; edgeExists {
						if edge, edgeOk := edgeEntity.(*Edge[E]); edgeOk {
							targetId := edge.TargetId
							if !visited[targetId] {
								stack = append(stack, targetId)
							}
						}
					}
				}
			}
		}
	}
}

func (g *Graph[N, E]) BFSAll() Map[string, *Graph[N, E]] {
	g.Lock.RLock()
	defer g.Lock.RUnlock()

	if g.Nodes.IsEmpty() {
		return Map[string, *Graph[N, E]]{}
	}

	visited := make(map[string]bool, len(g.Nodes))
	components := Map[string, *Graph[N, E]]{}
	componentIndex := 0

	list.New()

	for _, node := range g.Nodes {
		if !visited[node.Id] {
			component := g.bfsComponent(node.Id, visited)
			if len(component) > 0 {
				components[componentIndex] = component
				componentIndex++
			}
		}
	}

	return components
}

// DFSAll performs DFS on all connected components and returns a map
// where each key is the component index and value is the DFS traversal order
func (g *Graph[N, E]) DFSAll() Map[int, Stream[string]] {
	g.Lock.RLock()
	defer g.Lock.RUnlock()

	if g.Nodes.IsEmpty() {
		return Map[int, Stream[string]]{}
	}

	visited := make(map[string]bool, len(g.Nodes))
	components := make(Map[int, Stream[string]])
	componentIndex := 0

	for _, node := range g.Nodes {
		if !visited[node.Id] {
			component := g.dfsComponent(node.Id, visited)
			if len(component) > 0 {
				components[componentIndex] = component
				componentIndex++
			}
		}
	}

	return components
}

func (g *Graph[N, E]) bfsComponent(startNodeId string, visited map[string]bool) []string {
	result := make([]string, 0)
	queue := make([]string, 0, len(g.Nodes))

	queue = append(queue, startNodeId)
	visited[startNodeId] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		result = append(result, current)
		if nodeEntity, exists := g.ExistingEntities[current]; exists {
			if node, ok := nodeEntity.(*Node[N]); ok {
				for _, edgeId := range node.OutgoingEdges {
					if edgeEntity, edgeExists := g.ExistingEntities[edgeId]; edgeExists {
						if edge, edgeOk := edgeEntity.(*Edge[E]); edgeOk {
							targetId := edge.TargetId
							if !visited[targetId] {
								visited[targetId] = true
								queue = append(queue, targetId)
							}
						}
					}
				}
			}
		}
	}

	return result
}

// dfsComponent performs DFS for a single connected component
func (g *Graph[N, E]) dfsComponent(startNodeId string, visited map[string]bool) []string {
	result := make([]string, 0)
	stack := make([]string, 0, len(g.Nodes))

	stack = append(stack, startNodeId)

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[current] {
			continue
		}

		visited[current] = true
		result = append(result, current)

		if nodeEntity, exists := g.ExistingEntities[current]; exists {
			if node, ok := nodeEntity.(*Node[N]); ok {
				for i := len(node.OutgoingEdges) - 1; i >= 0; i-- {
					edgeId := node.OutgoingEdges[i]
					if edgeEntity, edgeExists := g.ExistingEntities[edgeId]; edgeExists {
						if edge, edgeOk := edgeEntity.(*Edge[E]); edgeOk {
							targetId := edge.TargetId
							if !visited[targetId] {
								stack = append(stack, targetId)
							}
						}
					}
				}
			}
		}
	}

	return result
}

// RemoveEdge removes an edge by ID and updates all related references
// Returns true if the edge was found and removed, false otherwise
func (g *Graph[N, E]) RemoveEdge(edgeId string) bool {
	if edgeId == "" {
		return false
	}

	g.Lock.Lock()
	defer g.Lock.Unlock()

	// Check if edge exists
	edgeEntity, exists := g.ExistingEntities[edgeId]
	if !exists {
		return false
	}

	edge, ok := edgeEntity.(*Edge[E])
	if !ok {
		return false
	}

	// Store source and target IDs before removal
	sourceId := edge.SourceId
	targetId := edge.TargetId

	// Remove edge from ExistingEntities map
	delete(g.ExistingEntities, edgeId)

	// Remove edge from Edges slice
	g.removeEdgeFromSlice(edgeId)

	// Update source node's outgoing edges
	if sourceEntity, sourceExists := g.ExistingEntities[sourceId]; sourceExists {
		if sourceNode, sourceOk := sourceEntity.(*Node[N]); sourceOk {
			sourceNode.Lock.Lock()
			sourceNode.OutgoingEdges = g.removeFromStringSlice(sourceNode.OutgoingEdges, edgeId)
			sourceNode.Lock.Unlock()
		}
	}

	// Update target node's incoming edges
	if targetEntity, targetExists := g.ExistingEntities[targetId]; targetExists {
		if targetNode, targetOk := targetEntity.(*Node[N]); targetOk {
			targetNode.Lock.Lock()
			targetNode.IngoingEdges = g.removeFromStringSlice(targetNode.IngoingEdges, edgeId)
			targetNode.Lock.Unlock()
		}
	}

	// Update graph's modified time
	now := time.Now()
	g.Modified = &now

	return true
}

// RemoveNode removes a node by ID and all its connected edges
// Returns true if the node was found and removed, false otherwise
func (g *Graph[N, E]) RemoveNode(nodeId string) bool {
	if nodeId == "" {
		return false
	}

	g.Lock.Lock()
	defer g.Lock.Unlock()

	node, ok := g.ExistingEntities[nodeId].(*Node[N])
	if !ok {
		return false
	}

	edgesToRemove := make(Stream[string], 0, len(node.OutgoingEdges)+len(node.IngoingEdges))
	for _, edgeId := range node.OutgoingEdges {
		edgesToRemove = append(edgesToRemove, edgeId)
	}

	for _, edgeId := range node.IngoingEdges {
		if !edgesToRemove.Contains(edgeId) {
			edgesToRemove = append(edgesToRemove, edgeId)
		}
	}

	for _, edgeId := range edgesToRemove {
		if edgeEntity, edgeExists := g.ExistingEntities[edgeId]; edgeExists {
			if edge, edgeOk := edgeEntity.(*Edge[E]); edgeOk {
				// Update the other node's edge references
				otherNodeId := ""
				if edge.SourceId == nodeId {
					otherNodeId = edge.TargetId
				} else if edge.TargetId == nodeId {
					otherNodeId = edge.SourceId
				}

				if otherNodeId != "" && otherNodeId != nodeId {
					if otherEntity, otherExists := g.ExistingEntities[otherNodeId]; otherExists {
						if otherNode, otherOk := otherEntity.(*Node[N]); otherOk {
							otherNode.Lock.Lock()
							otherNode.OutgoingEdges = g.removeFromStringSlice(otherNode.OutgoingEdges, edgeId)
							otherNode.IngoingEdges = g.removeFromStringSlice(otherNode.IngoingEdges, edgeId)
							otherNode.Lock.Unlock()
						}
					}
				}
			}
		}

		delete(g.ExistingEntities, edgeId)
		g.removeEdgeFromSlice(edgeId)
	}

	delete(g.ExistingEntities, nodeId)

	g.removeNodeFromSlice(nodeId)

	now := time.Now()
	g.Modified = &now

	return true
}

func (g *Graph[N, E]) RemoveNodeSafe(nodeId string) bool {
	if nodeId == "" {
		return false
	}

	g.Lock.Lock()
	defer g.Lock.Unlock()

	if _, ok := g.ExistingEntities[nodeId].(*Node[N]); !ok {
		return false
	}

	edgesToRemove := make([]string, 0)

	for _, edge := range g.Edges {
		if edge.SourceId == nodeId || edge.TargetId == nodeId {
			edgesToRemove = append(edgesToRemove, edge.Id)
		}
	}

	// Remove all connected edges
	for _, edgeId := range edgesToRemove {
		// Use the existing RemoveEdge method but without the outer lock
		g.removeEdgeInternal(edgeId)
	}

	// Remove node from ExistingEntities map
	delete(g.ExistingEntities, nodeId)

	// Remove node from Nodes slice
	g.removeNodeFromSlice(nodeId)

	// Update graph's modified time
	now := time.Now()
	g.Modified = &now

	return true
}

// RemoveMultipleEdges removes multiple edges by their IDs efficiently
// Returns the number of edges successfully removed
func (g *Graph[N, E]) RemoveMultipleEdges(edgeIds []string) int {
	if len(edgeIds) == 0 {
		return 0
	}

	g.Lock.Lock()
	defer g.Lock.Unlock()

	removedCount := 0
	nodeUpdates := make(map[string]*Node[N]) // Track nodes that need edge list updates

	// Process all edge removals first
	for _, edgeId := range edgeIds {
		if edgeEntity, exists := g.ExistingEntities[edgeId]; exists {
			if edge, ok := edgeEntity.(*Edge[E]); ok {
				// Track affected nodes
				if sourceEntity, sourceExists := g.ExistingEntities[edge.SourceId]; sourceExists {
					if sourceNode, sourceOk := sourceEntity.(*Node[N]); sourceOk {
						nodeUpdates[edge.SourceId] = sourceNode
					}
				}
				if targetEntity, targetExists := g.ExistingEntities[edge.TargetId]; targetExists {
					if targetNode, targetOk := targetEntity.(*Node[N]); targetOk {
						nodeUpdates[edge.TargetId] = targetNode
					}
				}

				// Remove edge
				delete(g.ExistingEntities, edgeId)
				g.removeEdgeFromSlice(edgeId)
				removedCount++
			}
		}
	}

	// Update all affected nodes' edge lists in batch
	edgeIdMap := make(map[string]bool, len(edgeIds))
	for _, edgeId := range edgeIds {
		edgeIdMap[edgeId] = true
	}

	for _, node := range nodeUpdates {
		node.Lock.Lock()
		node.OutgoingEdges = g.filterStringSlice(node.OutgoingEdges, edgeIdMap)
		node.IngoingEdges = g.filterStringSlice(node.IngoingEdges, edgeIdMap)
		node.Lock.Unlock()
	}

	if removedCount > 0 {
		now := time.Now()
		g.Modified = &now
	}

	return removedCount
}

// RemoveMultipleNodes removes multiple nodes by their IDs efficiently
// Returns the number of nodes successfully removed
func (g *Graph[N, E]) RemoveMultipleNodes(nodeIds []string) int {
	if len(nodeIds) == 0 {
		return 0
	}

	g.Lock.Lock()
	defer g.Lock.Unlock()

	nodeIdMap := make(map[string]bool, len(nodeIds))
	for _, nodeId := range nodeIds {
		nodeIdMap[nodeId] = true
	}

	removedNodeCount := 0
	edgesToRemove := make([]string, 0)

	// Collect all edges that need to be removed
	for _, edge := range g.Edges {
		if nodeIdMap[edge.SourceId] || nodeIdMap[edge.TargetId] {
			edgesToRemove = append(edgesToRemove, edge.Id)
		}
	}

	// Remove all affected edges
	for _, edgeId := range edgesToRemove {
		delete(g.ExistingEntities, edgeId)
		g.removeEdgeFromSlice(edgeId)
	}

	// Remove all specified nodes
	for _, nodeId := range nodeIds {
		if _, exists := g.ExistingEntities[nodeId]; exists {
			delete(g.ExistingEntities, nodeId)
			g.removeNodeFromSlice(nodeId)
			removedNodeCount++
		}
	}

	if removedNodeCount > 0 {
		now := time.Now()
		g.Modified = &now
	}

	return removedNodeCount
}

// Helper methods for efficient slice operations

// removeEdgeFromSlice removes an edge from the edges slice
func (g *Graph[N, E]) removeEdgeFromSlice(edgeId string) {
	for i, edge := range g.Edges {
		if edge.Id == edgeId {
			// Efficient removal by swapping with last element
			g.Edges[i] = g.Edges[len(g.Edges)-1]
			g.Edges = g.Edges[:len(g.Edges)-1]
			break
		}
	}
}

// removeNodeFromSlice removes a node from the nodes slice
func (g *Graph[N, E]) removeNodeFromSlice(nodeId string) {
	for i, node := range g.Nodes {
		if node.Id == nodeId {
			// Efficient removal by swapping with last element
			g.Nodes[i] = g.Nodes[len(g.Nodes)-1]
			g.Nodes = g.Nodes[:len(g.Nodes)-1]
			break
		}
	}
}

// removeFromStringSlice removes a string from a slice
func (g *Graph[N, E]) removeFromStringSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			// Efficient removal by swapping with last element
			slice[i] = slice[len(slice)-1]
			return slice[:len(slice)-1]
		}
	}
	return slice
}

// filterStringSlice removes all strings that exist in the exclusion map
func (g *Graph[N, E]) filterStringSlice(slice []string, exclusionMap map[string]bool) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if !exclusionMap[item] {
			result = append(result, item)
		}
	}
	return result
}

// removeEdgeInternal removes an edge without acquiring locks (for internal use)
func (g *Graph[N, E]) removeEdgeInternal(edgeId string) bool {
	edgeEntity, exists := g.ExistingEntities[edgeId]
	if !exists {
		return false
	}

	edge, ok := edgeEntity.(*Edge[E])
	if !ok {
		return false
	}

	sourceId := edge.SourceId
	targetId := edge.TargetId

	delete(g.ExistingEntities, edgeId)
	g.removeEdgeFromSlice(edgeId)

	// Update connected nodes
	if sourceEntity, sourceExists := g.ExistingEntities[sourceId]; sourceExists {
		if sourceNode, sourceOk := sourceEntity.(*Node[N]); sourceOk {
			sourceNode.Lock.Lock()
			sourceNode.OutgoingEdges = g.removeFromStringSlice(sourceNode.OutgoingEdges, edgeId)
			sourceNode.Lock.Unlock()
		}
	}

	if targetEntity, targetExists := g.ExistingEntities[targetId]; targetExists {
		if targetNode, targetOk := targetEntity.(*Node[N]); targetOk {
			targetNode.Lock.Lock()
			targetNode.IngoingEdges = g.removeFromStringSlice(targetNode.IngoingEdges, edgeId)
			targetNode.Lock.Unlock()
		}
	}

	return true
}

func (g *Graph[N, E]) String() string {
	g.Lock.RLock()
	defer g.Lock.RUnlock()

	return fmt.Sprint(
		strings.Repeat("*", 20), "\n",
		"GraphId: ", g.Id, "\n",
		"NodeCount: ", g.Nodes.Len(), "\n",
		"EdgeCount: ", g.Edges.Len(), "\n",
		strings.Repeat("*", 20),
	)
}
