package bot

import (
	"archive/tar"
	"cmp"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/ulikunitz/xz"
)

func (b *Bot) createBackup() (err error) {
	now := time.Now()
	backupFileName := fmt.Sprintf("%s.%s", b.backupFile, now.Format("2006-01-02_15-04-05"))
	log.Printf("creating backup: %s", backupFileName)
	defer func() {
		if err != nil {
			log.Printf("error while creating backup: %v", err)
		} else {
			log.Printf("backup %s created successfully", backupFileName)
		}
	}()

	backupFile := path.Join(b.backupDir, backupFileName)
	_, err = b.db.ExecContext(b.ctx, "VACUUM INTO ?", backupFile)
	if err != nil {
		return err
	}
	return nil
}

func (b *Bot) compressBackups() (err error) {
	log.Println("compressing backups")
	defer func() {
		if err != nil {
			log.Printf("error while compressing backups: %v", err)
		} else {
			log.Println("backups compressed successfully")
		}
	}()

	entries, err := os.ReadDir(b.backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}
	infos := make([]fs.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("failed to get file info: %w", err)
		}
		if !strings.HasPrefix(info.Name(), b.backupFile) {
			continue
		}
		infos = append(infos, info)
	}

	if len(infos) == 0 {
		log.Println("no backups found to compress")
		return nil
	}

	slices.SortFunc(infos, func(a, b fs.FileInfo) int {
		return cmp.Compare(a.ModTime().UnixNano(), b.ModTime().UnixNano())
	})

	first := infos[0].ModTime().Format("2006-01-02_15-04-05")
	last := infos[len(infos)-1].ModTime().Format("2006-01-02_15-04-05")

	backupFile := path.Join(b.backupDir, fmt.Sprintf("backups.%s_%s.tar.xz", last, first))
	f, err := os.Create(backupFile)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer f.Close()

	xzw, err := xz.NewWriter(f)
	if err != nil {
		return fmt.Errorf("failed to create xz writer: %w", err)
	}
	defer xzw.Close()

	tw := tar.NewWriter(xzw)
	defer tw.Close()

	for _, info := range infos {
		if info.IsDir() {
			continue
		}
		err := func() error {
			filePath := path.Join(b.backupDir, info.Name())
			file, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to open backup file: %w", err)
			}
			defer file.Close()

			hdr, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return fmt.Errorf("failed to create tar header: %w", err)
			}
			hdr.Name = info.Name()
			hdr.ModTime = info.ModTime()
			hdr.Typeflag = tar.TypeReg

			err = tw.WriteHeader(hdr)
			if err != nil {
				return fmt.Errorf("failed to write tar header: %w", err)
			}

			if _, err := io.Copy(tw, file); err != nil {
				return fmt.Errorf("failed to copy file to tar: %w", err)
			}
			return nil
		}()
		if err != nil {
			return fmt.Errorf("failed to add file to tar: %w", err)
		}
	}

	for _, info := range infos {
		if info.IsDir() {
			continue
		}
		filePath := path.Join(b.backupDir, info.Name())
		err := os.Remove(filePath)
		if err != nil {
			return fmt.Errorf("failed to remove backup file: %w", err)
		}
	}

	return nil
}
