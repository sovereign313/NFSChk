when trying to access an NFS Mount that is hung, the entire shell (and any process) that tries to access that mount point hangs.  This tool runs a goroutine (thread) to try to access the NFS share.
if the share is hung, and the goroutine doesn't return in a certain amount of time, then it will try to unmount and remount the share.  Also logs the steps along the way.
