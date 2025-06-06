package snapshot

import (
	"os"
	"slices"

	"github.com/containerd/containerd/v2/core/mount"
	"github.com/pkg/errors"
)

func (lm *localMounter) Mount() (string, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.mounts == nil && lm.mountable != nil {
		mounts, release, err := lm.mountable.Mount()
		if err != nil {
			return "", err
		}
		lm.mounts = mounts
		lm.release = release
	}

	if !lm.forceRemount && len(lm.mounts) == 1 && lm.mounts[0].Type == "nullfs" {
		ro := slices.Contains(lm.mounts[0].Options, "ro")
		if !ro {
			return lm.mounts[0].Source, nil
		}
	}

	dir, err := os.MkdirTemp("", "buildkit-mount")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	if err := mount.All(lm.mounts, dir); err != nil {
		os.RemoveAll(dir)
		return "", errors.Wrapf(err, "failed to mount %s: %+v", dir, lm.mounts)
	}
	lm.target = dir
	return dir, nil
}

func (lm *localMounter) Unmount() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.target != "" {
		if err := mount.Unmount(lm.target, 0); err != nil {
			return err
		}
		os.RemoveAll(lm.target)
		lm.target = ""
	}

	if lm.release != nil {
		return lm.release()
	}

	return nil
}
