# Structuring your code

Sometimes, when we use `protoconf` we will want to write helpers functions and global constants that we might want to include in multiple configs. We can define those in `.pinc` files.

`.pinc` files are starlark code which doesn't produce configs (doesn't evaluate the `main()` function).

Example:

```python
"""
file: ./src/helpers.pinc
"""
PROTOCONF_VERSION="0.1.3"

def format_name(person):
    # assumes `person` is a proto message that have `first_name` and `last_name`
    return "%s %s" % (person.first_name, person.last_name)
```

We can now load the variables and functions in this file to another `starlark` file (`pinc`, `.pconf` or `.mpconf`)

```python
load("//helpers.pinc", "PROTOCONF_VERSION", "format_name")
```