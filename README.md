# annotated-go-macaron

- macaron 项目源码注解. 

## Macaron 简介:

- [macaron - github](https://github.com/go-macaron/macaron)
- [macaron - 官网](https://go-macaron.com/)
- Macaron: 一款具有高生产力和模块化设计的 Go Web 框架
- 项目灵感: 来源于[martini](https://github.com/go-martini/martini) 框架


## 源码版本:

- [macaron-1.1.8](./macaron-1.1.8)
    - 当前最新版本. `release: on 27 Aug 2016`
    - 代码行数统计: 4483 (含测试代码: 2006行)

## 源码结构:

```

-> % tree ./macaron-1.1.8 -L 2

./macaron-1.1.8
├── README.md
├── context.go
├── context_test.go
├── fixtures
│   ├── basic
│   ├── basic2
│   ├── custom_funcs
│   └── symlink
├── logger.go
├── logger_test.go
├── macaron.go                  // 项目全局入口
├── macaron_test.go
├── recovery.go
├── recovery_test.go
├── render.go
├── render_test.go
├── response_writer.go
├── response_writer_test.go
├── return_handler.go
├── return_handler_test.go
├── router.go
├── router_test.go
├── static.go
├── static_test.go
└── tree.go

5 directories, 20 files


```








