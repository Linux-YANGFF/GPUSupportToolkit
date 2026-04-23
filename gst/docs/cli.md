# GST CLI Reference

GST CLI is a command-line tool for parsing and analyzing GPU log files.

## Installation

If you have a deb/rpm package installed:
```bash
# Debian/Ubuntu
sudo dpkg -i gst_*.deb

# RHEL/CentOS/Fedora
sudo rpm -i gst-*.rpm
```

Or build from source:
```bash
go build -o bin/gst-cli ./cmd/cli
```

## Usage

```
gst-cli [options] -parse <file>
```

## Options

| Flag | Description |
|:----|:------------|
| `-parse <file>` | Parse the specified log file (required) |
| `-search <keyword>` | Search for keyword(s) in log (space-separated for multiple) |
| `-time <start,end>` | Search by time range in microseconds (format: startUs,endUs) |
| `-top <N>` | Show top N slowest frames (default: 10) |
| `-funcs` | Show function call statistics |
| `-shader` | Show shader compilation statistics |
| `-export <format>` | Export results (txt, csv, or json) |
| `-output <file>` | Output file path (default: stdout) |
| `-help` | Show help information |

## Commands

### Parse Log File

Analyze a GPU log file and display summary statistics:

```bash
gst-cli -parse /path/to/log.trace
```

Output includes:
- Detected log format type
- Total frame count
- FPS (frames per second)
- First/last frame time

### Search Keywords

Search for specific API calls or keywords in the log:

```bash
# Single keyword
gst-cli -search "glDrawElements" -parse /path/to/log.trace

# Multiple keywords (space-separated)
gst-cli -search "glDrawElements glFlush" -parse /path/to/log.trace
```

### Time Range Search

Find API calls within a specific time range (in microseconds):

```bash
gst-cli -time "1000,50000" -parse /path/to/log.trace
```

### Frame Analysis

Display the slowest frames ranked by total time:

```bash
# Top 10 slowest frames (default)
gst-cli -top 10 -parse /path/to/log.trace

# Top 20 slowest frames
gst-cli -top 20 -parse /path/to/log.trace
```

Output includes:
- Frame number
- Total time (in microseconds and milliseconds)
- API call count
- Slowest API call in each frame

### Function Statistics

Show aggregated statistics for all API functions:

```bash
gst-cli -funcs -parse /path/to/log.trace
```

Output table:
- Function name
- Call count
- Total time (microseconds)
- Average time (microseconds)

### Shader Statistics

Display shader compilation statistics:

```bash
gst-cli -shader -parse /path/to/log.trace
```

Output includes compile count and total compile time per shader type.

### Export Results

Export parsed data in various formats:

```bash
# JSON export
gst-cli -export json -output result.json -parse /path/to/log.trace

# CSV export
gst-cli -export csv -output result.csv -parse /path/to/log.trace

# TXT export
gst-cli -export txt -output result.txt -parse /path/to/log.trace
```

## Combined Examples

```bash
# Full analysis: parse, show top 20 frames, and function stats
gst-cli -top 20 -funcs -parse /path/to/log.trace

# Search while parsing
gst-cli -search "glShaderSource" -parse /path/to/log.trace

# Export with custom output
gst-cli -export json -output analysis.json -top 50 -parse /path/to/log.trace
```

## Exit Codes

- `0`: Success
- `1`: Error (file not found, parse failure, etc.)