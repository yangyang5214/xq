
雪球组合调仓抓取

### Usage

```
./xq --help
从 cookies.txt 与组合列表拉取雪球组合的调仓历史，可选拉取单次调仓详情。

Usage:
  xq [flags]

Flags:
      --cookies-file string      Get cookies.txt LOCALLY 导出的 cookies.txt 路径 (default "cookies.txt")
  -c, --count int                每页条数(1-50) (default 20)
  -f, --cubes-file string        组合列表文件路径，每行一个 symbol，支持 # 注释 (default "cubes.txt")
  -h, --help                     help for xq
  -p, --page int                 调仓历史页码 (default 1)
  -t, --to string                收件人邮箱（多个用逗号分隔）
  -w, --weight-threshold float   比例变化阈值(%)，超过此值才发邮件提醒 (default 5)
```

### Others

Get cookies.txt LOCALLY 插件

https://github.com/kairi003/Get-cookies.txt-LOCALLY 
