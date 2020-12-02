### 作业


> 我们在数据库操作的时候，比如 `dao` 层中当遇到一个 `sql.ErrNoRows` 的时候，是否应该 `Wrap` 这个 `error`，抛给上层。为什么？应该怎么做请写出代码

***这个问题想问的是,查询的时候要对err进行一次判断,看是否为sql.ErrNoRows?***

以下所有基于这个理解所写

先看看这个Err的定义[https://golang.org/pkg/database/sql/#ErrNoRows](https://golang.org/pkg/database/sql/#ErrNoRows)

ErrNoRows is returned by Scan when QueryRow doesn't return a row. In such a case, QueryRow returns a placeholder *Row value that defers this error until a Scan.

```golang
var ErrNoRows = errors.New("sql: no rows in result set")
```

ErrNoRows其实就是一个`sentinel error`。针对是否`wrap`这个error进行讨论。

#### 不单独处理`sql.ErrNoRows`，即直接返回

```golang

// querySome 基于某种条件的查询
func QuerySome() (int, error) {
    var some int
    if err := db.QueryRow("select some from foo where bar = '一路向西'").Scan(&some); err != nil {
        return 0, errors.Wrap(err, "xxxx")
    }

    return some, int
}

func fooBar() error{
    _, err := querySome()
    if err != nil {
        // xxxx
        return 
    }

    // do something
    return 
}
```

`fooBar`中拿到的`error`就是原始的没有stack的error

- sql.ErrNoRows是无业务意义的

如检查某个object是否存在的情况，wrap不wrap无所谓,直接消耗掉，降级处理

```
if _, err := QuerySome(); err != nil {
    if !errors.Is(err, sql.ErrNoRows){
        return
    }
    // doing something
    err = nil
}
```
此时log打出来的日志,几时包含了堆栈信息，但是并没有任何实际的作用。

- sql.ErrNoRows是有业务意义的

`GET /path/to/object/:id` 的这种场景, service直接调用了`dao`层的方法，来判断资源是否存在,资源不存在，返回`404`

```
var foo *Bar
if foo, err := findFooById(id); err != nil {
    if errors.Is(err, sql.ErrNoRows) {
        log.Warning("query some got error: %+v", errors.UnWrap(err))
        err = status.Error(codes.NotFound, "not found")
        return
    }
    // 返回错误5xx
    return 
}
```

无论这个`sql.ErrNoRows`是否是有业务含义的，在上层代码都需要进行一次额外的判断。显得啰嗦。并且，这个被`wrap`住的err虽然带上了上下文堆栈,但是并没有实际的价值。

#### 单独处理`sql.ErrNoRows`。

```golang
type Foo struct {
    Bar string
}

// GetFoo 根据主键查找
func GetFoo(id int64) (*Foo, error) {
    var foo Foo
    if err := db.QueryRow("select * from bar where id = $1", id).Scan(&foo.Bar); err != nil {
        if err == sql.ErrNoRows {
            return &foo, nil
        }
        return nil, errors.Wrap(err, "get foo error")
    }
}

func fooBar() error{
    foo, err := GetFoo(1)
    if err != nil {
        // xxxx
        return 
    }

    // do something
    fmt.Println(foo.Bar)
    return 
}
```

对于上层调用代码来说,判断err的地方变得更加简洁了。但是带来的问题,foo的值，当`no rows`时候的返回,真正的值就变成了它类型的零值。对于对象的零值，还好。但是对于如查某个int字段的值,它的零值也就是0,也是一个有意义的值，无法区分。

#### 总结

`dao`层,也就是date access object，即是数据访问对象。而数据源有很多种,可能是`mysql`, `redis`, `mongo`等。redis中的not found，则是`redis.Nil`。如果直接`Wrap(sql.ErrNoRows)`返回，对于上层调用来说，势必会增加负担。若dao层的数据源有多种，还需要根据自身逻辑判断多种not found。如果经过特殊处理，返回类型的零值。可能在某些场景下会发生歧义，产生逻辑错误。逻辑上的错误远远比多写几行代码严重得多。

综上优缺点。认为在`dao`层应该不需要对`sql.ErrNoRows`进行特殊处理，对所有的err都wrap,还需带上相关的上下文信息。想必调用方对在什么场景下的查询导致了not found的原因更加关注吧。并且，提供一个判断是否为`sql.ErrNoRows`的公开方法来进行判断。

```golang
package dao

// 可能的数据获取出错类型
var noFoundErrs = []error{sql.ErrNoRows, redis.Nil}

// IsNoRows 判断是否为not found记录
func IsNoRows(err error) bool{
    for _, item := range noFoundErrs {
        if errors.Is(err, item) {
            return true
        }
    }
    return false
}

```