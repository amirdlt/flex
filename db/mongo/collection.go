package mongo

import (
	"context"
	. "github.com/amirdlt/flex/util"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
)

type Collection struct {
	*mongo.Collection
}

type SearchConstraints struct {
	Skip          int
	Limit         int
	OnlyCount     bool
	Regex         primitive.Regex
	Contains      bool
	CaseSensitive bool
	Fields        bson.M
}

func (c *Collection) DeleteAll(ctx context.Context) (*mongo.DeleteResult, error) {
	return c.DeleteMany(ctx, bson.M{})
}

func (c *Collection) Search(ctx context.Context, sc SearchConstraints, v any) (int, error) {
	var filter bson.D
	var orTerms bson.A

	stringChecker := func(v any) bson.M {
		value := v.(string)
		var condition bson.M
		if sc.Regex.Pattern != "" {
			condition = bson.M{
				"$regex": sc.Regex,
			}
		} else if sc.CaseSensitive {
			if sc.Contains {
				condition = bson.M{
					"$regex": primitive.Regex{Pattern: ".*" + value + ".*", Options: ""},
				}
			} else {
				condition = bson.M{"$eq": value}
			}
		} else if sc.Contains {
			condition = bson.M{
				"$regex": primitive.Regex{
					Pattern: ".*" + value + ".*", Options: "i",
				},
			}
		} else {
			condition = bson.M{
				"$regex": primitive.Regex{Pattern: "^" + value + "$", Options: "i"},
			}
		}

		return condition
	}

	if queries, exist := sc.Fields["__queries__"].([]any); !exist {
		var filterBuilder func(k string, v M)
		filterBuilder = func(k string, v M) {
			for kk, vv := range v {
				var key string
				if k == "" {
					key = kk
				} else {
					key = k[1:] + "." + kk
				}

				switch reflect.ValueOf(vv).Kind() {
				case reflect.String:
					filter = append(filter, bson.E{Key: key, Value: stringChecker(vv)})
				case reflect.Array, reflect.Slice:
					values := vv.([]any)
					switch len(values) {
					case 0:
						continue
					case 1:
						filter = append(filter, bson.E{Key: key, Value: stringChecker(vv.([]any)[0])})
					default:
						var terms bson.A
						for _, v := range values {
							terms = append(terms, bson.M{key: stringChecker(v)})
						}

						orTerms = append(orTerms, bson.M{"$or": terms})
					}
				default:
					filterBuilder(k+"."+kk, vv.(M))
				}
			}
		}
		filterBuilder("", sc.Fields)
	} else {
		for _, condition := range queries {
			var terms bson.A
			for _, v := range condition.([]any) {
				vv := v.(M)
				terms = append(terms, bson.M{vv["key"].(string): stringChecker(vv["value"])})
			}

			orTerms = append(orTerms, bson.M{"$or": terms})
		}
	}

	filter = append(filter, bson.E{Key: "$and", Value: orTerms})

	if sc.OnlyCount {
		if count, err := c.CountDocuments(ctx, filter); err != nil {
			return -1, err
		} else {
			return int(count), nil
		}
	} else {
		cursor, err := c.Find(ctx, filter, options.Find().
			SetSkip(int64(sc.Skip)).SetLimit(int64(sc.Limit)))
		if nil != err {
			return -1, err
		}

		if err := cursor.All(ctx, v); nil != err {
			return -1, err
		}

		return reflect.ValueOf(v).Elem().Len(), nil
	}
}

func (c *Collection) Exists(ctx context.Context, filter bson.M) (bool, error) {
	if err := c.FindOne(ctx, filter).Err(); err == nil {
		return true, nil
	} else if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	} else {
		return false, err
	}
}

func (c *Collection) GetById(ctx context.Context, id, v any, idKey ...string) error {
	_idKey := "_id"
	if len(idKey) > 0 {
		_idKey = idKey[0]
	}

	return c.FindOne(ctx, bson.M{_idKey: id}).Decode(v)
}
