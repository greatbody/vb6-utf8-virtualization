# Windows File System Transparent Encoding Conversion Proxy

This project is a Windows-based utility (service/daemon) that creates a virtual filesystem layer to automatically convert file encodings on the fly. It is specifically designed to bridge the gap between legacy GB2312/GBK encoded files (e.g., from old VB6 projects) and modern UTF-8 based applications or AI agents.

## 🌟 Features

- **Transparent Virtual Drive**: Mounts a virtual drive (e.g., `Z:`) that proxies a physical directory (e.g., `C:\Source`).
- **Dynamic Transcoding**: Automatically detects and converts GB2312/GB18030 files to UTF-8 when read, and converts UTF-8 back to GB18030 when written.
- **Process Filtering**: Only applies transcoding for specific applications (e.g., `a.exe`, `notepad.exe`). Other processes see the raw physical files.
- **File Type Whitelisting**: Only transcodes specific extensions (e.g., `.txt`, `.bas`, `.frm`, `.cls`, `.vbp`).
- **BOM Handling**: Automatically handles and cleans UTF-8 Byte Order Marks (BOM).

## 🛠 Prerequisites

To build and run this project, you need:

1.  **Windows 10 or 11**.
2.  **Dokan Driver**: Download and install the [Dokan library](https://github.com/dokan-dev/dokany/releases) (v2.x is recommended).
3.  **Go (Golang)**: Installed and configured on your Windows machine (1.20+ recommended).

## 🚀 Getting Started

### 1. Clone the Project
Since you are developing on macOS, you can zip this directory and move it to your Windows environment or use Git.

### 2. Build for Windows

On your **Windows** machine, open a command prompt or PowerShell in the project directory and run:

```batch
build.bat
```

Or manually:

```powershell
go build -o utf8proxy.exe main.go
```

### 3. Configuration

Create or edit `config.json` in the same directory as the executable. You can use `config.json.example` as a template:

```json
{
    "physical_path": "C:\\Path\\To\\Your\\Legacy\\Project",
    "mount_point": "Z:",
    "allowed_processes": ["a.exe", "notepad.exe", "vscode.exe"],
    "allowed_extensions": [".txt", ".csv", ".bas", ".frm", ".cls", ".vbp", ".log", ".ini"]
}
```

- `physical_path`: The real folder containing your files.
- `mount_point`: The letter for the virtual drive.
- `allowed_processes`: List of process names (case-insensitive) that should see the converted UTF-8 content.
- `allowed_extensions`: List of file extensions to transcode.

### 4. Running the Proxy

```powershell
.\utf8proxy.exe -config config.json
```

The program will stay open and maintain the mount. To stop, press `Ctrl+C`.

## 📂 Project Structure

- `main.go`: Application entry point and Dokan mounting logic.
- `internal/transcoder/`: Core encoding detection and conversion engine.
- `internal/vfs/`: Dokan filesystem implementation and filtering logic.
- `internal/config/`: Configuration file management.

## ⚠️ Important Limitations

- **Random Access (Seek)**: Because encoding conversion changes byte lengths (e.g., 2-byte Chinese characters become 3-byte UTF-8), random seeking is not natively supported for transcoded files. The proxy reads the entire file into memory during the first access to ensure consistency.
- **File Size Reporting**: The virtual drive reports the size of the UTF-8 version of the file, which may differ from the actual size on disk.
- **Performance**: Large files (hundreds of MBs) in the whitelist will be loaded into memory. For non-whitelisted files or processes, there is zero overhead.

## 📜 License

This project is licensed under the MIT License.
The Dokan library itself and the Go bindings used are subject to their respective licenses (LGPL/BSD).
