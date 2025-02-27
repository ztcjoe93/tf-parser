# tf-parser: simple parser for terraform files
This is a simple parser for terraform files.

# Setting up
To run:
```shell
$ go run . -f filepath_to_tf -b resource_block_type
```

To test:
```shell
$ go test . -v 
```

To test with coverage:
```shell
$ go test . -v -coverprofile cover.out
$ go tool cover -html cover.out -o cover.html
```

To compile into a binary and execute:
```
$ go build . && ./tf-parser
```

# To use

| flag | description | default |
| --- | --- | --- |
| `-v` | Enable verbose logging | `false` |
| `-f` | Path to terraform file to parse | `` |
| `-b` | Resource block to look for | `default` |
| `-o` | Output path for extracted resource | `extracted.tf` |

| command | description |
| --- | --- |
| `list` | Default command, lists resources of specified block |
| `extract` | Extracts target resources into a separate terraform file |

# How it works
## Parsing logic
When parsing `.tf` files, the parser systematically parses each block of resource and stores the starting and ending line numbers.  

The parsing logic uses a simple syntax validation strategy, where the vital `{` and `}` are given a value of +1 and -1 respectively, where having the value resolve to 0 will indicate that it has come to the end of the resource block.  

In a simple example:  
```main.tf
1 locals {
2   my_variable = "here"
3   custom_var = {
4       some_other_var = "some_value"
5   }
6 }
```

- In line 1, the parser detects a `{` symbol which indicates the start of a resource block, parses `locals` as the resource type, and increments the value to `1`
- The parser iteratively goes through each line to look for `{` or `}`
- At line 3, the parser detects another `{` symbol, and increments the value to `2`
- At line 5, the closing bracket for `custom_var` is found and the value is decremented to `1`
- At line 6, the closing bracket for the `locals` resource is found and value is decremented to `0`, which denotes the end of the resource block as well

## Mapping logic
Each block is mapped into a hash map, where the type.name is mapped to its lines. These lines are then used to extract resources out to separate files, or for sorting lexicographically. The only exception is the `locals` resource block, as there is no naming to these blocks.  

```
{
  "locals": [
	 [line_contents],
	 [line_contents],
	 ...
  ],
  "resource": {
	  "aws_vpc.this": [line_contents],
	  "aws_vpc_dhcp_options_association.this": [line_contents],
	  ...
  }
}
```
