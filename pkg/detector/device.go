package detector

import "strings"

func DetectDeviceType(userAgent string) string {
	ua := strings.ToLower(userAgent)

	botKeywords := []string{"bot", "crawler", "spider", "scraper", "curl", "wget"}
	for _, keyword := range botKeywords {
		if strings.Contains(ua, keyword) {
			return "bot"
		}
	}

	mobileKeywords := []string{"mobile", "android", "iphone", "ipod", "blackberry", "windows phone"}
	for _, keyword := range mobileKeywords {
		if strings.Contains(ua, keyword) {
			return "mobile"
		}
	}

	tabletKeywords := []string{"tablet", "ipad"}
	for _, keyword := range tabletKeywords {
		if strings.Contains(ua, keyword) {
			return "tablet"
		}
	}

	if strings.Contains(ua, "mozilla") || strings.Contains(ua, "windows") || strings.Contains(ua, "macintosh") {
		return "desktop"
	}

	return "unknown"
}

func GetClientIP(remoteAddr, xForwardedFor, xRealIP string) string {
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if xRealIP != "" {
		return xRealIP
	}

	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		return remoteAddr[:idx]
	}

	return remoteAddr
}
