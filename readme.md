## go-Lru存储
1. 支持设置最大容量
2. 10s触发一次检查，删掉超过容量的数据

## 使用

### 引用

```
go get github.com/kangkang66/go-lru
```

### 初始化
```
cache := NewCache(10)
```

### 存储

```
cache.StoreOrUpdate("config_key", 1, "com.app.name", []byte(`{"name":"zhangsan"}`))
```


### 获取

```
value,err := cache.Get("config_key", 1, "com.app.name")
```

### 获取当前的keys
```
keys := cache.AllKeys()
```

