package assert

import "fmt"

func Assert(predicate bool, message string) {
    if !predicate {
        panic(message)
    }
}

func Assert_eq(item1 interface{}, item2 interface{}, message string) {
    if item1 != item2 {
        panic(fmt.Sprintf("item 1: %v, item 2: %v: %s", item1, item2, message))
    }
}

func Assert_ne(item1 interface{}, item2 interface{}, message string) {
    if item1 == item2 {
        panic(fmt.Sprintf("item 1: %v, item 2: %v: %s", item1, item2, message))
    }
}

// i dont wanna do proper error handling for this
func Expect(err error, message string) {
    if err != nil {
        err = fmt.Errorf("FATAL: %s: %s", message, err)
        panic(err)
    }
}
