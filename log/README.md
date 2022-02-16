# log
写文件日志和日志中心日志  

# 日志中心的使用
## 通过以下步骤启动日志中心  
1. 设置启动日志中心
   可以使用以下方法启动日志中心  
   - 调用`log.InitRemoteLog`
   - 调用`log.SetLogModuleName`
   - 调用`log.SetRemoteLogLevel`
   - 调用`log.SetLog`
1. 调用`ADD agent /opt/logagent/agent`把agent打包到image里
1. (optional)通过环境变量`LOG_AGENT_PATH`设置agent所在的位置(如果是`/opt/logagent/agent`则可以不设)
1. (optional)通过环境变量`LOG_AGENT_KAFKA`设置kafka主机和端口

## 环境变量
### `LOG_AGENT_PATH`
设置agent所在的位置，为空时则为`/opt/logagent/agent`

### `LOG_AGENT_KAFKA`
设置kafka 的主机端口，多个主机使用英文逗号分隔，为空时则去consul 里去取  

### `LOG_TRACE_FILE`
设置调试信息的文件，为空时不打印调试信息  

