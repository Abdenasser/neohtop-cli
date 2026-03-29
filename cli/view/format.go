package view

import "fmt"

// FormatBytes converts bytes to human-readable format
func FormatBytes(bytes uint64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(bytes)
	idx := 0

	for value >= 1024 && idx < len(units)-1 {
		value /= 1024
		idx++
	}

	return fmt.Sprintf("%.1f %s", value, units[idx])
}

// FormatBytesCompact converts bytes to a compact form (no space, e.g. "1.2G")
func FormatBytesCompact(bytes uint64) string {
	units := []string{"B", "K", "M", "G", "T"}
	value := float64(bytes)
	idx := 0

	for value >= 1024 && idx < len(units)-1 {
		value /= 1024
		idx++
	}

	if idx == 0 {
		return fmt.Sprintf("%d%s", bytes, units[idx])
	}
	if value >= 100 {
		return fmt.Sprintf("%.0f%s", value, units[idx])
	}
	if value >= 10 {
		return fmt.Sprintf("%.1f%s", value, units[idx])
	}
	return fmt.Sprintf("%.1f%s", value, units[idx])
}

// FormatMemorySize formats bytes as GB
func FormatMemorySize(bytes uint64) string {
	gb := float64(bytes) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.1f GB", gb)
}

// FormatPercentage formats a float as percentage
func FormatPercentage(value float32) string {
	return fmt.Sprintf("%.1f%%", value)
}

// FormatUptime formats seconds into days/hours/minutes
func FormatUptime(seconds uint64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// FormatRuntime formats process runtime in a compact way
func FormatRuntime(seconds uint64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	}
	if seconds < 86400 {
		return fmt.Sprintf("%dh %dm", seconds/3600, (seconds%3600)/60)
	}
	return fmt.Sprintf("%dd %dh", seconds/86400, (seconds%86400)/3600)
}

// Truncate truncates a string to maxLen with ellipsis
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return s[:maxLen]
	}
	return s[:maxLen-1] + "…"
}

// PadRight pads a string with spaces to the given width
func PadRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + spaces(width-len(s))
}

// PadLeft pads a string with spaces on the left
func PadLeft(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return spaces(width-len(s)) + s
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}
