package git

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
)

// WritePacketLine encodes a string into Git packet-line protocol format (4-byte hex length + payload).
func WritePacketLine(w io.Writer, msg string) error {
	length := len(msg) + 4
	_, err := fmt.Fprintf(w, "%04x%s", length, msg)
	return err
}

// WritePacketFlush writes the Git packet-line flush token (0000).
func WritePacketFlush(w io.Writer) error {
	_, err := fmt.Fprint(w, "0000")
	return err
}

// HandleInfoRefs handles the Git Smart HTTP ref advertisement handshake (GET /info/refs?service=...).
func HandleInfoRefs(w http.ResponseWriter, r *http.Request, diskPath, service string) {
	if service != "git-upload-pack" && service != "git-receive-pack" {
		http.Error(w, "Unsupported service", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-advertisement", service))
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)

	// 1. Initial packet: # service=<service>\n
	_ = WritePacketLine(w, fmt.Sprintf("# service=%s\n", service))
	// 2. Flush packet
	_ = WritePacketFlush(w)

	// 3. Subprocess: git <service_name> --stateless-rpc --advertise-refs <diskPath>
	gitSubCmd := strings.TrimPrefix(service, "git-")
	cmd := exec.Command("git", gitSubCmd, "--stateless-rpc", "--advertise-refs", diskPath)
	cmd.Stdout = w

	_ = cmd.Run()
}

// HandleServiceRPC handles Git Smart HTTP RPC requests (POST /git-upload-pack and POST /git-receive-pack).
func HandleServiceRPC(w http.ResponseWriter, r *http.Request, diskPath, service string) {
	if service != "git-upload-pack" && service != "git-receive-pack" {
		http.Error(w, "Unsupported service", http.StatusBadRequest)
		return
	}

	var reqBody io.ReadCloser = r.Body
	defer reqBody.Close()

	// Handle GZIP decompression if client compressed the payload
	if r.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "Failed to decompress gzip body", http.StatusBadRequest)
			return
		}
		defer gzReader.Close()
		reqBody = gzReader
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-result", service))
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)

	gitSubCmd := strings.TrimPrefix(service, "git-")
	cmd := exec.Command("git", gitSubCmd, "--stateless-rpc", diskPath)
	cmd.Stdin = reqBody
	cmd.Stdout = w

	_ = cmd.Run()
}
