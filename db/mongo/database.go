package mongo

import (
	. "github.com/amirdlt/flex/util"
	"go.mongodb.org/mongo-driver/mongo"
)

type Database struct {
	collections Map[string, Collection]
	*mongo.Database
}

func (d Database) GetCollection(name string) Collection {
	if col, exist := d.collections[name]; exist {
		return col
	}

	d.collections[name] = Collection{
		Collection: d.Collection(name),
	}

	return d.collections[name]
}
