**代码已经迁移至 https://github.com/acicn/deployer2**

# deployer2

`deployer` 的船新版本

## 使用方法

`deployer2` 会从 Jenkins 任务的环境变量 `$JOB_NAME` 获取信息

任务名 `hello-world.test` 会自动生成参数 `--image hello-world --profile test`

```
Usage of ./deployer2:
  -cpu value
    	指定 CPU 配额，格式为 "MIN:MAX"，单位为 m (千分之一核心)
  -image string
    	镜像名
  -manifest string
    	指定描述文件 (default "deployer.yml")
  -mem value
    	指定 MEM 配额，格式为 "MIN:MAX"，单位为 Mi (兆字节)
  -profile string
    	指定环境名
  -skip-deploy
    	跳过部署流程
  -workload value
    	指定目标工作负载，可以指定多次，格式为 "CLUSTER/NAMESPACE/TYPE/NAME[/CONTAINER]"
```

## 集群预置文件 (Preset)

**一般情况下，集群预置文件由管理员负责配置，一般用户不需要关心**

集群预置文件用来描述某个特定 Kubernetes 集群的配置信息，包括使用的镜像仓库，镜像仓库拉取密钥，镜像仓库推送配置，连接集群所需的 Kubernetes 配置等信息

集群预置文件需要保存在 `$HOME/.deployer2/preset-DEMO.yml` 位置，其中 `DEMO` 为集群名，之后的 `--workload` 参数会用到集群名。

集群预置文件内容如下

```yaml
# 镜像仓库地址，可以包含组织名
registry: ccr.ccs.tencentyun.com/acicn
# 工作负载注解，注意这个注解是在 Deployment, Statefulset 等控制器级别，不在 Pod 级别
annotations:
    net.guoyk.autodown/lease: 128h
# 镜像拉取秘钥，Kubernetes 集群要想从镜像拉取镜像，要使用哪个 Secret
imagePullSecrets:
    - qcloudregistry
# 默认资源限制
resource:
  cpu: 100:200 # CPU 单位为毫核心，冒号后可以使用 - 表示无限制
  mem: 200:- # MEM 单位为兆，冒号后可以使用 - 表示无限制
# 集群的 Kubeconfig 文件内容，以 YAML 格式
kubeconfig:
  # xxxx
# 推送镜像所需的 .docker/config.json 文件内容，以 YAML 格式
dockerconfig:
  auths: # ...
```

## 项目清单文件 (Manifest)

项目清单文件 `deployer.yml` 一般保存在项目代码根路径下 

项目清单文件可以包含多套环境配置 (Profile)

其中，`default` 环境为默认环境，其他环境 (`dev`, `test` 等等) 如果缺乏某些值的时候，会从 `default` 环境配置中获取默认值

文件内容如下

```yaml
# 必须设置，标记版本为 2
version: 2
# 默认环境配置 `default`
default:
  build:
    - #...
  package:
    - #...
# 其他任意多个环境配置
dev:
  # ...
test:
  # ...
```

## 环境配置 (Profile)

如上文所说，`deployer.yml` 可以包含多个环境配置，每套环境配置由以下字段组成，如果字段缺失，会从 `default` 环境中获取值并填充

```yaml
# 构建脚本，数组格式，本质为 Bash 脚本
build:
  - npm install
  - npm run build:{{.Vars.env}} # build 脚本允许使用模板语言，此处从 vars 中引用 env 变量
# 打包脚本，数组格式，本质为 Dockerfile 文件
package:
  - FROM acicn/node:{{.Vars.node_version}} # package 也允许使用模板语言，此处从 vars 中引用 node_version 变量
# 资源申请与限制
resource:
  cpu: 200:2000 # CPU 资源配置，单位为毫核，前者为申请值，后者为限制值
  mem: 200:2000 # 内存 资源配置，单位为兆，前者为申请值，后者为限制值
  # 上述所有限制值，可以用 - 表示无限制，比如
  # mem: 200:-
# 健康检查
check:
  port: 8080 # 健康检查端口，默认为 800
  path: /health/check # 健康检查路径，如果没有设置路径，则关闭健康检查
  delay: 60 # 健康检查起始时间，默认为 60 秒，如果项目需要更长时间来完成启动，可以增加该值
  interval: 15 # 健康检查周期，默认为 15 秒
  success:   1 # 多少次健康检查成功后，判定项目已经成功启动，默认为 1
  failure:   2 # 多少次健康检查失败后，判定项目失败，默认为 2
  timeout:   5 # 健康检查接口超时时间，默认为 5 秒
# 自定义参数，可以用来渲染 build 和 package 字段，一般用例下，只在 default 环境中填写 build 和 package 字段，其他环境均使用 vars 参数来修改不同环境下的渲染结果
vars:
  env: test
  node_version: 12
```

### 使用 Docker 镜像作为 build 环境

如果要使用 Docker 镜像中的 `bash` 作为 `build` 脚本执行环境，而非使用当前主机的 `bash`，需要在默认环境或者其他环境中设置参数 `builder`

```yaml
builder:
  image: acicn/node-builder:12
  # 缓存组，默认为 default，具有相同缓存组的任务，会使用相同的文件夹
  cacheGroup: biz
  caches:
    - /root/.npm
```

`deployer2` 会强制执行以下内容

1. 把当前工作目录映射到容器内的 `/workspace` 目录
2. 映射 `caches` 字段的目录到主机 `$HOME/.deployer2-builder-cache/biz` 的子目录下
2. 把 `build` 脚本渲染后映射到容器内的 `/deployer2-build-script.sh` 文件下
3. 在 `/deployer2-build-script.sh` 脚本之前添加 `cd /workspace` 确保脚本能在容器内的 `/workspace` 目录下执行
4. 在 `/deployer2-build-script.sh` 脚本末尾添加 `chown -R XXX:XXX /workspace` 将 `/workspace` 也就是当前工作目录的权限改回到宿主机用户
5. 在容器内执行 `/deployer2-build-script.sh` 命令

### 完整示例

以下示例仅用于完整展示 `deployer2` 的功能

* `$HOME/.deployer2/preset-k8s-prod.yml` 文件内容

```yaml
registry: ccr.ccs.tencentyun.com/helo
kubeconfig:
  # ....
dockerconfig:
  # ...
```

* `deployer.yml` 文件内容:

```yaml
version: 2

# 默认环境配置
default:
  # 默认给项目 CPU 半个核心到两个核心，内存 256MB 到 2000MB
  resource:
    cpu: 500:2000
    mem: 256:2000
  # 默认启用健康检查
  check:
    port: 3000
    path: /check
  # 构建脚本
  build:
    - npm install
    - npm run build:{{.Vars.env}} # 不同环境，执行不同的 npm run build:xxxx 命令
  # 打包脚本
  package:
    - FROM acicn/node:{{.Vars.node_version}} # 不同环境，使用不同的 node_version
    - WORKDIR /work
    - ADD . .
    - ENV MINIT_MAIN "npm start"

# test 环境
test:
  # 继承 default 的 resource, build, package 字段
  check:
    path: "" # 此处将健康检查 path 覆盖为空字符串，则关闭健康检查
  # 使用 vars 字段对 build 和 package 渲染结果进行控制
  vars:
    env: test # 此处会导致 build 字段第二行渲染为 "npm run build:test"
    node_version: 12 # 此处会导致 package 字段首行渲染为 "FROM acicn/node:12"

# prod 环境
prod:
  # 继承 default 的 check, build, package 字段
  resource:
    # CPU 配置继承 default
    # 内存 配置修改为申请 200兆，不限制内存大小
    mem: 200:-
  # 使用 vars 字段对 default 的 build 和 package 渲染结果进行控制
  vars:
    env: production # 此处会导致 build 字段第二行渲染为 "npm run build:production"
    node_version: 10 # 此处会导致 package 字段首行渲染为 "FROM acicn/node:10"
```

* 创建 Jenkins 任务 `hello-world.prod`

  任务内容为如下 Shell 脚本

    ```shell script
    deployer2 --cluster k8s-prod/hello/deployment/hello-world
    ```

执行该命令，`deployer2` 命令行工具会

1. 从 `$HOME/.deployer2/preset-k8s-prod.yml` 文件读取预置文件 (Preset)
2. 从 Jenkins 任务名，选择 `hello-world` 为镜像名，选取 `prod` 为环境名
3. 选择 `prod` 环境的自定变量 `vars`
4. 渲染 `build` 字段为如下内容

    ```shell script
    npm install
    npm run build:production
    ```
   
5. 执行上述脚本
6. 渲染 `package` 字段为如下内容

    ```dockerfile
    FROM acicn/node:10
    WORKDIR /work
    ADD . .
    ENV MINIT_MAIN "npm start"
    ```
7. 从 `--workload` 参数得知，要更新 `k8s-prod` 集群的，`hello` 命名空间下的，名字叫 `hello-world` 的 `Deployment` 类型的工作负载

8. 推送镜像 `ccr.ccs.tencentyun.com/hello/hello-world:prod-build-X`，并调用 `kubectl` 为工作负载修改镜像名，资源限制和健康检查配置

## 许可证

Guo Y.K., MIT License
