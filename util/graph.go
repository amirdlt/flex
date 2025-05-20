package util

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type EntityId = string

type Entity[T any] struct {
	Lock *sync.RWMutex

	Id      EntityId
	Type    string
	Data    T
	Created time.Time
}

type Edge[N, E any] struct {
	Entity[E]

	from *Node[N, E]
	to   *Node[N, E]
}

type Node[N, E any] struct {
	Entity[N]

	ingoing  Stream[*Edge[N, E]]
	outgoing Stream[*Edge[N, E]]
}

type Graph[N, E any] struct {
	Entity[*Graph[N, E]]

	Edges Map[EntityId, *Edge[N, E]]
	Nodes Map[EntityId, *Node[N, E]]

	EdgeKeyGen func(from, to *Node[N, E], data E) EntityId
	NodeKeyGen func(n N) EntityId
}

func NewGraph[N, E any](nodeIdGen func(N) any, edgeIdGen ...func(*Node[N, E], *Node[N, E], E) any) *Graph[N, E] {
	if nodeIdGen == nil {
		nodeIdGen = func(n N) any {
			h, _ := Hash(n)
			return h
		}
	}

	n := time.Now()
	if len(edgeIdGen) == 0 {
		edgeIdGen = []func(*Node[N, E], *Node[N, E], E) any{
			func(from, to *Node[N, E], _ E) any {
				return fmt.Sprint(from.Id, "-->", n.UnixNano(), "-->", to)
			},
		}
	}

	return &Graph[N, E]{
		Entity: Entity[*Graph[N, E]]{
			Lock:    &sync.RWMutex{},
			Id:      GenerateUUID("graph", n.UnixNano()),
			Type:    "graph",
			Data:    nil,
			Created: n,
		},
		Nodes: Map[EntityId, *Node[N, E]]{},
		Edges: Map[EntityId, *Edge[N, E]]{},
		NodeKeyGen: func(n N) EntityId {
			return GenerateUUID("node", nodeIdGen(n))
		},
		EdgeKeyGen: func(from, to *Node[N, E], data E) EntityId {
			return GenerateUUID("edge", edgeIdGen[0](from, to, data))
		},
	}
}

func (g *Graph[N, E]) AddNode(data N) EntityId {
	id := g.NodeKeyGen(data)

	g.Lock.Lock()
	if node := g.Nodes[id]; node != nil {
		node.Data = data
	} else {
		n := &Node[N, E]{
			Entity: Entity[N]{
				Id:      id,
				Data:    data,
				Type:    "node",
				Created: time.Now(),
				Lock:    &sync.RWMutex{},
			},
		}

		g.Nodes[id] = n
	}

	g.Lock.Unlock()

	return id
}

func (g *Graph[N, E]) AddEdge(from, to EntityId, data E) EntityId {
	f, t := g.Nodes[from], g.Nodes[to]
	if f == nil || t == nil {
		panic("invalid id for from/to vertices")
	}

	id := g.EdgeKeyGen(f, t, data)

	g.Lock.Lock()
	if edge := g.Edges[id]; edge != nil {
		edge.Data = data
	} else {
		e := &Edge[N, E]{
			Entity: Entity[E]{
				Lock:    &sync.RWMutex{},
				Id:      id,
				Data:    data,
				Type:    "edge",
				Created: time.Now(),
			},
			from: f,
			to:   t,
		}

		f.ingoing = append(f.ingoing, e)
		t.outgoing = append(t.outgoing, e)
		g.Edges[id] = e
	}

	g.Lock.Unlock()

	return id
}

func (g *Graph[N, E]) RemoveEdge(id EntityId) bool {
	g.Lock.Lock()
	defer g.Lock.Unlock()

	e := g.Edges[id]
	if e == nil {
		return false
	}

	e.from.outgoing = e.from.outgoing.Remove(e)
	e.to.ingoing = e.to.outgoing.Remove(e)

	return true
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

func (g *Graph[N, E]) GetNode(data N) *Node[N, E] {
	return g.GetNodeById(g.NodeKeyGen(data))
}

func (g *Graph[N, E]) GetNodeById(id EntityId) *Node[N, E] {
	g.Lock.RLock()
	n := g.Nodes[id]
	g.Lock.RUnlock()

	return n
}

func (g *Graph[N, E]) GetEdge(from, to EntityId, data E) *Edge[N, E] {
	return g.GetEdgeById(g.EdgeKeyGen(g.Nodes[from], g.Nodes[to], data))
}

func (g *Graph[N, E]) GetEdgeById(id EntityId) *Edge[N, E] {
	g.Lock.RLock()
	e := g.Edges[id]
	g.Lock.RUnlock()

	return e
}

func (g *Graph[N, E]) ForEachEdge(consumer func(edge *Edge[N, E])) {
	g.Lock.RLock()
	g.Edges.Values().ForEach(consumer)
	g.Lock.RUnlock()
}

func (g *Graph[N, E]) ForEachNode(consumer func(node *Node[N, E])) {
	g.Lock.RLock()
	g.Nodes.Values().ForEach(consumer)
	g.Lock.RUnlock()
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
