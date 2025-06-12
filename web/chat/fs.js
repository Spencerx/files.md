// Sandboxed filesystem implementation using File System Access API
// Operates within a user-selected directory sandbox

const fsImpl = (() => {
    let sandboxHandle = null;
    let fileHandles = new Map(); // fd -> {handle, writable?, position}
    let nextFd = 3; // start after stdin/stdout/stderr
    let isInitialized = false;

    const encoder = new TextEncoder();
    const decoder = new TextDecoder();

    const enosys = () => {
        const err = new Error("not implemented");
        err.code = "ENOSYS";
        return err;
    };

    const enoent = () => {
        const err = new Error("no such file or directory");
        err.code = "ENOENT";
        return err;
    };

    const eacces = () => {
        const err = new Error("permission denied");
        err.code = "EACCES";
        return err;
    };

    const eexist = () => {
        const err = new Error("file exists");
        err.code = "EEXIST";
        return err;
    };

    // Initialize sandbox directory
    const initSandbox = async () => {
        if (!('showDirectoryPicker' in window)) {
            throw new Error("File System Access API not supported");
        }

        if (!isInitialized) {
            sandboxHandle = await window.showDirectoryPicker({
                mode: 'readwrite',
                startIn: 'documents'
            });
            isInitialized = true;
            console.log('Filesystem sandbox initialized:', sandboxHandle.name);
        }
        return sandboxHandle;
    };

    // Navigate to handle by path within sandbox
    const getHandleByPath = async (path, options = {}) => {
        await initSandbox();

        // Normalize path - remove leading/trailing slashes, handle current dir
        const normalizedPath = path.replace(/^\/+|\/+$/g, '').replace(/\/+/g, '/');
        if (!normalizedPath || normalizedPath === '.') {
            return sandboxHandle;
        }

        const parts = normalizedPath.split('/');
        let currentHandle = sandboxHandle;

        // Navigate through directories
        for (let i = 0; i < parts.length - 1; i++) {
            const part = parts[i];
            if (part === '.' || part === '') continue;

            try {
                currentHandle = await currentHandle.getDirectoryHandle(part);
            } catch {
                if (options.create) {
                    currentHandle = await currentHandle.getDirectoryHandle(part, { create: true });
                } else {
                    throw enoent();
                }
            }
        }

        const fileName = parts[parts.length - 1];
        if (fileName === '.' || fileName === '') {
            return currentHandle;
        }

        // Get file handle
        try {
            return await currentHandle.getFileHandle(fileName, options);
        } catch {
            if (options.create) {
                return await currentHandle.getFileHandle(fileName, { create: true });
            }
            throw enoent();
        }
    };

    // Get directory handle by path
    const getDirHandleByPath = async (path, options = {}) => {
        await initSandbox();

        const normalizedPath = path.replace(/^\/+|\/+$/g, '').replace(/\/+/g, '/');
        if (!normalizedPath || normalizedPath === '.') {
            return sandboxHandle;
        }

        const parts = normalizedPath.split('/');
        let currentHandle = sandboxHandle;

        for (const part of parts) {
            if (part === '.' || part === '') continue;

            try {
                currentHandle = await currentHandle.getDirectoryHandle(part);
            } catch {
                if (options.create) {
                    currentHandle = await currentHandle.getDirectoryHandle(part, { create: true });
                } else {
                    throw enoent();
                }
            }
        }

        return currentHandle;
    };

    return {
        // Initialize the sandbox (call this first)
        async init() {
            return await initSandbox();
        },

        writeSync(fd, buf) {
            if (fd === 1 || fd === 2) { // stdout/stderr
                const text = decoder.decode(buf);
                console.log(text);
                return buf.length;
            }

            const fdInfo = fileHandles.get(fd);
            if (!fdInfo || !fdInfo.writable) return 0;

            // For real sync operation, we'd need to buffer writes
            // This is a placeholder that returns the length
            return buf.length;
        },

        write(fd, buf, offset, length, position, callback) {
            if (offset !== 0 || length !== buf.length || position !== null) {
                callback(enosys());
                return;
            }

            if (fd === 1 || fd === 2) {
                const text = decoder.decode(buf);
                console.log(text);
                callback(null, buf.length);
                return;
            }

            const fdInfo = fileHandles.get(fd);
            if (!fdInfo) {
                callback(enoent());
                return;
            }

            (async () => {
                try {
                    if (!fdInfo.writable) {
                        fdInfo.writable = await fdInfo.handle.createWritable();
                    }
                    await fdInfo.writable.write(buf);
                    callback(null, buf.length);
                } catch (err) {
                    callback(err);
                }
            })();
        },

        read(fd, buffer, offset, length, position, callback) {
            const fdInfo = fileHandles.get(fd);
            if (!fdInfo) {
                callback(enoent());
                return;
            }

            (async () => {
                try {
                    const file = await fdInfo.handle.getFile();
                    const arrayBuffer = await file.arrayBuffer();
                    const data = new Uint8Array(arrayBuffer);

                    const start = position !== null ? position : fdInfo.position || 0;
                    const end = Math.min(start + length, data.length);
                    const bytesToRead = Math.max(0, end - start);

                    if (bytesToRead > 0) {
                        buffer.set(data.subarray(start, start + bytesToRead), offset);
                        if (position === null) {
                            fdInfo.position = start + bytesToRead;
                        }
                    }

                    callback(null, bytesToRead);
                } catch (err) {
                    callback(err);
                }
            })();
        },

        open(path, flags, mode, callback) {
            (async () => {
                try {
                    const isWrite = flags & 1; // O_WRONLY or O_RDWR
                    const isCreate = flags & 64; // O_CREAT

                    const handle = await getHandleByPath(path, { create: isCreate });
                    const fd = nextFd++;

                    fileHandles.set(fd, {
                        handle: handle,
                        writable: null,
                        position: 0,
                        isWrite: isWrite
                    });

                    callback(null, fd);
                } catch (err) {
                    callback(err);
                }
            })();
        },

        close(fd, callback) {
            const fdInfo = fileHandles.get(fd);
            if (fdInfo) {
                (async () => {
                    try {
                        if (fdInfo.writable) {
                            await fdInfo.writable.close();
                        }
                        fileHandles.delete(fd);
                        callback(null);
                    } catch (err) {
                        callback(err);
                    }
                })();
            } else {
                callback(null);
            }
        },

        stat(path, callback) {
            (async () => {
                try {
                    // Try as file first
                    try {
                        const handle = await getHandleByPath(path);
                        const file = await handle.getFile();

                        callback(null, {
                            size: file.size,
                            mtime: new Date(file.lastModified),
                            atime: new Date(file.lastModified),
                            ctime: new Date(file.lastModified),
                            isFile: () => true,
                            isDirectory: () => false,
                            mode: 0o644
                        });
                        return;
                    } catch {}

                    // Try as directory
                    const dirHandle = await getDirHandleByPath(path);
                    callback(null, {
                        size: 0,
                        mtime: new Date(),
                        atime: new Date(),
                        ctime: new Date(),
                        isFile: () => false,
                        isDirectory: () => true,
                        mode: 0o755
                    });
                } catch (err) {
                    callback(err);
                }
            })();
        },

        fstat(fd, callback) {
            const fdInfo = fileHandles.get(fd);
            if (!fdInfo) {
                callback(enoent());
                return;
            }

            (async () => {
                try {
                    const file = await fdInfo.handle.getFile();
                    callback(null, {
                        size: file.size,
                        mtime: new Date(file.lastModified),
                        atime: new Date(file.lastModified),
                        ctime: new Date(file.lastModified),
                        isFile: () => true,
                        isDirectory: () => false,
                        mode: 0o644
                    });
                } catch (err) {
                    callback(err);
                }
            })();
        },

        readdir(path, callback) {
            (async () => {
                try {
                    const dirHandle = await getDirHandleByPath(path);
                    const entries = [];

                    for await (const [name, handle] of dirHandle.entries()) {
                        entries.push(name);
                    }

                    callback(null, entries);
                } catch (err) {
                    callback(err);
                }
            })();
        },

        mkdir(path, perm, callback) {
            (async () => {
                try {
                    await getDirHandleByPath(path, { create: true });
                    callback(null);
                } catch (err) {
                    callback(err);
                }
            })();
        },

        rmdir(path, callback) {
            (async () => {
                try {
                    const parentPath = path.substring(0, path.lastIndexOf('/')) || '.';
                    const dirName = path.substring(path.lastIndexOf('/') + 1);

                    const parentHandle = await getDirHandleByPath(parentPath);
                    await parentHandle.removeEntry(dirName, { recursive: false });
                    callback(null);
                } catch (err) {
                    callback(err);
                }
            })();
        },

        unlink(path, callback) {
            (async () => {
                try {
                    const parentPath = path.substring(0, path.lastIndexOf('/')) || '.';
                    const fileName = path.substring(path.lastIndexOf('/') + 1);

                    const parentHandle = await getDirHandleByPath(parentPath);
                    await parentHandle.removeEntry(fileName);
                    callback(null);
                } catch (err) {
                    callback(err);
                }
            })();
        },

        rename(from, to, callback) {
            // File System Access API doesn't support rename directly
            // Would need to copy + delete
            callback(enosys());
        },

        chmod(path, mode, callback) { callback(null); },
        chown(path, uid, gid, callback) { callback(null); },
        fchmod(fd, mode, callback) { callback(null); },
        fchown(fd, uid, gid, callback) { callback(null); },
        fsync(fd, callback) { callback(null); },
        ftruncate(fd, length, callback) { callback(enosys()); },
        lchown(path, uid, gid, callback) { callback(null); },
        link(path, link, callback) { callback(enosys()); },
        lstat(path, callback) { this.stat(path, callback); },
        readlink(path, callback) { callback(enosys()); },
        symlink(path, link, callback) { callback(enosys()); },
        truncate(path, length, callback) { callback(enosys()); },
        utimes(path, atime, mtime, callback) { callback(null); }
    };
})();

// Usage: Call fsImpl.init() first to set up the sandbox
if (typeof module !== 'undefined') {
    module.exports = fsImpl;
} else {
    window.fsImpl = fsImpl;
}