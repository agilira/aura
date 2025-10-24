## Aura

---
![Aura](images/aura.png)

*Aura* is a simple *&* fast build tool written in golang

the tool require `aura.yaml` file in your project to run

**Commands:**

- `aura build -t <targets>` - run build targets
- `aura list` - show available targets
- `aura init --template <type>` - create new project
- `aura clean` - remove build artifacts
- `aura watch -t <targets>` - watch files and rebuild
- `aura validate` - check config file

**Variables:**

- you can declare a variable using this syntax

```yaml
vars:
  CC: "gcc"
  CFLAGS: "-Wall -o2"

```

```yaml
vars:
  GO: "go"
  GFLAGS: "build -o"
  OUT: "aura2.exe"
```

- get a variable or env variable

```yaml
$CC or ${CC}
```

- Built in variables :
	* `$cwd`       get current working directory
	* `$@`         get current target name
	* `$TIMESTAMP` get current time

**Targets:**

- Declare a target

```yaml
targets:
  test:
    run:
      - "echo this is test"
      
  build:
    onerror: "This is a Custom Error" # custom error
    run:
      - "echo Target Name: $@"
      - "$GO $GFLAGS $OUT"
    deps:
      - test

  continue_on_error: false # if command fails exit for this target only
```


- `continue_on_error`  if command fails exit for this target only
- it can be declared as a global to all the targets

**Prologue & Epilogue**

- this will run always at the start
```yaml
prologue:
  run:
    - "echo Working in $cwd"
```

- this will run always at the end

```yaml
epilogue:
  run:
    - "echo Finished at $TIMESTAMP"
```

- they are targets too

---

*Includes:*

- You can include other files

```yaml

include:
  - "other_file.yaml"

```

*Project Templates:*

- Initialize new projects with templates

```bash
aura init --template go     # Go project
aura init --template rust   # Rust project  
aura init --template node   # Node.js project
aura init --template basic  # Basic C/C++ project
```

*Very Simple Example:*

```yaml
vars:
  GO: "go"
  FLAGS: "build -o"
  EXE: "aura2.exe"

targets:
  build:
    run:
      - "$GO $FLAGS $EXE"
  
  start:
    deps:
      - build
    run:
      - "$EXE -h"
```

*Output:*

```bash
PS I:\golang\Aura> aura build -t start
Building target: start
Dependency: build
go build -o aura2.exe
Usage of aura2.exe:
  -D string
        Working Directory (default ".")
  -c string
        Configuration file path (default "aura.yaml")
  -v    Enable verbose output
```

*Building:*

```bash
// linux
go env -w GOOS="linux"
go build

// windows
go env -w GOOS="windows"
go build

```

*Development:*

```bash
// run tests
go test ./...

// run with coverage
go test -cover ./...

// format code
go fmt ./...
```


