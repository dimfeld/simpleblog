Second Post
5/15/14 12:15AM -0500
second post, some tag

# This is the latest post

Ok...

````go
func readChan(c chan int) bool {
    logger.Println("try read channel")

    select {
    case a := <-c:
        fmt.Println("Received", a)
        return true

    default:
        fmt.Println("Channel not ready")
        return false, 5
    }
}
````