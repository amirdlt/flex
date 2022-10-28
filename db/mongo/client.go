package mongo

import (
	"context"
	. "github.com/amirdlt/flex/util"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

type Client struct {
	name      string
	databases map[string]Database
	*mongo.Client
}

type Clients map[string]Client

func (c Client) GetDatabase(name string) Database {
	if db, exist := c.databases[name]; exist {
		return db
	}

	c.databases[name] = Database{
		collections: Map[string, *Collection]{},
		Database:    c.Database(name),
	}

	return c.databases[name]
}

func (c Clients) AddClient(name, connectionUrl string) error {
	if _, exist := c[name]; exist {
		panic("client already exists")
	}

	if client, err := mongo.Connect(context.TODO(), options.Client().
		ApplyURI(connectionUrl)); err != nil {
		return err
	} else {
		c[name] = Client{
			databases: map[string]Database{},
			name:      name,
			Client:    client,
		}

		return nil
	}
}

func (c Clients) GetClient(name string) Client {
	if client, exist := c[name]; exist {
		return client
	}

	panic("invalid client name, please first all a client")
}

func (c Clients) ClearClient(name string) {
	if client, exist := c[name]; exist {
		if err := client.Client.Disconnect(context.Background()); err != nil {
			log.Println("could not disconnect client:", err)
		} else {
			delete(c, name)
		}
	}
}

func (c Clients) ClearAllClients() {
	for k := range c {
		c.ClearClient(k)
	}
}
