package util

import (
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

func newEntity[T any](typeV string, key any, data T) *Entity[T] {
	return &Entity[T]{
		Id:      GenerateUUID(typeV, key),
		Data:    data,
		Created: time.Now(),
		Lock:    &sync.RWMutex{},
	}
}

type Edge[T any] struct {
	*Entity[T]

	SourceId string `json:"source_ref" bson:"source_ref"`
	TargetId string `json:"target_ref" bson:"target_ref"`
}

type Node[T any] Entity[T]

type Graph[N, E any] struct {
	Entity[*Graph[N, E]]

	lock *sync.RWMutex

	Created  time.Time  `json:"created" bson:"created"`
	Modified *time.Time `json:"modified,omitempty" bson:"modified,omitempty"`

	Edges            Stream[*Edge[E]] `json:"edges,omitempty" bson:"edges,omitempty"`
	Nodes            Stream[*Node[N]] `json:"nodes,omitempty" bson:"nodes,omitempty"`
	ExistingEntities Map[string, any] `json:"-" bson:"-"`
}

func NewGraph[N, E any](key ...any) *Graph[N, E] {
	if len(key) == 0 {
		key = []any{time.Now()}
	}

	return &Graph[N, E]{
		Entity: Entity[*Graph[N, E]]{
			Id:   GenerateUUID("graph", key),
			Type: "graph",
			Data: nil,
		},
		ExistingEntities: Map[string, any]{},
		lock:             &sync.RWMutex{},
	}
}

func (g *Graph[N, E]) AddNode(data N, key any) string {
	id := GenerateUUID("node", key)

	g.lock.Lock()
	if node, exist := g.ExistingEntities[id].(*Node[N]); exist {
		node.Data = data
	} else {
		n := &Node[N]{
			Id:      id,
			Data:    data,
			Type:    "node",
			Created: time.Now(),
		}

		g.Nodes = append(g.Nodes, n)
		g.ExistingEntities[id] = n
	}

	g.lock.Unlock()

	return id
}

func (g *Graph[N, E]) AddEdge(from, to string, data E, key any) string {
	id := GenerateUUID("edge", key)

	g.lock.Lock()
	if edge, exist := g.ExistingEntities[id].(*Edge[E]); exist {
		edge.SourceId = from
		edge.TargetId = to
		edge.Data = data
	} else {
		e := &Edge[E]{
			Entity: Entity[E]{
				Id:      id,
				Data:    data,
				Type:    "relationship",
				Created: time.Now(),
			},
			SourceId: from,
			TargetId: to,
		}

		g.Edges = append(g.Edges, e)
		g.ExistingEntities[id] = e
	}

	g.lock.Unlock()

	return id
}

func (g *Graph[N, E]) AddNodeIfNotExist(data any, key any) string {
	id := GenerateUUID("node", key)

	g.lock.RLock()
	if g.ExistingEntities.ContainKey(id) {
		return id
	}

	g.lock.RUnlock()

	g.AddNode(data, key)

	return id
}

func (g *Graph[N, E]) AddEdgeIfNotExist(from, to string, data E, key any) string {
	id := GenerateUUID("edge", key)

	g.lock.RLock()
	if g.ExistingEntities.ContainKey(id) {
		return id
	}

	g.lock.RUnlock()

	g.AddEdge(from, to, data, key)

	return id
}

func (g *Graph[N, E]) Entities() Stream[any] {
	res := make([]any, len(g.Nodes)+len(g.Edges))

	var index int

	g.lock.RLock()
	for _, n := range g.Nodes {
		res[index] = *n
		index++
	}

	for _, e := range g.Edges {
		res[index] = *e
		index++
	}

	g.lock.RUnlock()

	return res
}

func (g *Graph[N, _]) GetNode(key any) Node[N] {
	id := GenerateUUID("node", key)

	g.lock.RLock()
	n := *g.ExistingEntities.Get(id).(*Node[N])
	g.lock.RUnlock()

	return n
}

func (g *Graph[_, E]) GetEdge(key any) Edge[E] {
	id := GenerateUUID("edge", key)

	g.lock.RLock()
	e := *g.ExistingEntities.Get(id).(*Edge[E])
	g.lock.RUnlock()

	return e
}

func (g *Graph[N, E]) Filter(filter func()) Stream[any] {
	if filter == nil {
		return g.ExistingEntities.Values()
	}

	return g.ExistingEntities.Values()
}
