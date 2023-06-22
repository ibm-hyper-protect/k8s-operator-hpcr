This sample creates a contract and a definition for an onprem pod running a busybox container with a volume mount to an attached data disk. The container will append to
a file on the data disk with every reboot and the expectation is that the size of the file increases across reboots. This proves that the data disk is really persistent across reboots.

Output looks like:

```text
Hello from busybox at Thu Jun 22 15:55:20 UTC 2023
Hello from busybox at Thu Jun 22 15:58:39 UTC 2023
Hello from busybox at Thu Jun 22 16:00:49 UTC 2023
```

on the attached logDNA.
