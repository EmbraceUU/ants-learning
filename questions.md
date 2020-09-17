# questions

## 00001

Q: 如果在retrieveWorker开始就加了锁, 那么应该只有一个协程可以进到这里, 那么blockingNum还有什么意义呢 ?

A: 没有理解cond条件锁的用法, cond.Wait()过程中会把lock释放, 并且阻塞在wait中, 这样其他协程就可以获得lock了.

## 00002

Q: CompareAndSwapInt32的用法是什么, 为什么用它来作为reboot的条件 ? 这么做, 有什么优点 ?

A: 有三个参数, 分别是是addr, old, new

```
if addr == old , return true
if addr == old, addr = new
```

代表了如果state为closed, 那么改为open, 如果原本为Open, 则返回false, 不改变state. 使用CompareAndSwapInt32让上面的逻辑变的简单和安全. 