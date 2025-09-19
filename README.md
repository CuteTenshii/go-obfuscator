# Go Obfuscator

A simple Go code obfuscator that renames variables, functions, and types to make the code harder to read while preserving its functionality.

## Features

- Renames variables, constants, functions, and types to random strings.
- Encodes string literals in base64. Decodes them at runtime with `base64.StdEncoding.DecodeString`, which makes it harder to read the original strings while keeping the code functional.
- Removes comments from the code

## Example

### Original Code

```go
package main

import "fmt"

func main() {
    // This is a sample function
    result := add(5, 3)
    fmt.Println("The result is:", result)
}

func add(a int, b int) int {
    return a + b
}
```

### Obfuscated Code

```go
package main

import (
    "encoding/base64"
    "fmt"
)

func main() {
    uscZPFcKDC := eXREHCOPOK(5, 3)
    fmt.Println(decodeBase64("VGhlIHJlc3VsdCBpczog"), uscZPFcKDC)
}

func eXREHCOPOK(aTAeSYNGNL int, fsSDEfgrkv int) int {
    return aTAeSYNGNL + fsSDEfgrkv
}

func decodeBase64(s string) string {
    decoded, err := base64.StdEncoding.DecodeString(s)
    if err != nil {
        panic(err)
    }
    return string(decoded)
}
```
