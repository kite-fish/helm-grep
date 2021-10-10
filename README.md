# helm-grep

A helm plugin :  release values searcher

## How to install

只支持helm3

### Using Helm plugin manager（>3.0.2）
```bash
helm plugin install https://github.com/phczheng/helm-grep
```



### From Source
#### Prerequisites
- GoLand >=1.14

Make sure you do not have a verison of helm-grep installed. You can remove it by running helm plugin uninstall grep
#### Installation Steps

```bash
git clone https://github.com/phczheng/helm-grep.git
cd helm-grep
make install
```

## How to use
```bash
helm grep -n default -r prometheus image
```

## Used libraries
	"gopkg.in/yaml.v3"

	"github.com/fatih/color"
	"github.com/mikefarah/yq/v4/pkg/yqlib"

	"github.com/urfave/cli/v2"
