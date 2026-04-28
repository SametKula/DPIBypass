package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"time"

	"github.com/getlantern/systray"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("DPI Bypass")
	systray.SetTooltip("macOS DPI Bypass Proxy")

	mStatus := systray.AddMenuItem("Durum: Çalışıyor", "Şu an proxy aktif")
	mStatus.Disable()

	mGithub := systray.AddMenuItem("Github", "Github repo sayfasını aç")
	go func() {
		<-mGithub.ClickedCh
		exec.Command("open", "https://github.com/sametkula/DPIBypass").Run()
	}()
	mGithub.SetTooltip("Github repo sayfasını aç")

	systray.AddSeparator()

	mAuthor := systray.AddMenuItem("Samet Kula", "")
	mAuthor.Disable()

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Kapat", "Uygulamayı kapat ve ayarları sıfırla")

	// 1. İşletim sistemi proxy ayarlarını otomatik yap
	setMacProxy(true)

	// 2. Arka planda proxy sunucusunu başlat
	go startProxyServer()

	// 3. Çıkış butonuna basıldığında
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

func onExit() {
	// Program kapanırken işletim sistemi proxy ayarlarını temizle (Kritik!)
	setMacProxy(false)
	fmt.Println("Uygulama kapatıldı, proxy ayarları eski haline getirildi.")
}

func setMacProxy(enable bool) {
	// Mac'te genelde Wi-Fi veya Ethernet kullanılır, ikisine de otomatik uyguluyoruz
	services := []string{"Wi-Fi", "Ethernet"}
	for _, service := range services {
		if enable {
			exec.Command("networksetup", "-setwebproxy", service, "127.0.0.1", "8080").Run()
			exec.Command("networksetup", "-setsecurewebproxy", service, "127.0.0.1", "8080").Run()
			exec.Command("networksetup", "-setwebproxystate", service, "on").Run()
			exec.Command("networksetup", "-setsecurewebproxystate", service, "on").Run()
		} else {
			exec.Command("networksetup", "-setwebproxystate", service, "off").Run()
			exec.Command("networksetup", "-setsecurewebproxystate", service, "off").Run()
		}
	}
}

func startProxyServer() {
	addr := "127.0.0.1:8080"
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Proxy başlatılamadı: %v", err)
	}
	fmt.Printf("macOS DPI Bypass Proxy %s üzerinde çalışıyor...\n", addr)

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleProxy(clientConn)
	}
}

func handleProxy(clientConn net.Conn) {
	defer clientConn.Close()

	// 1. Tarayıcıdan gelen HTTP CONNECT isteğini oku
	reader := bufio.NewReader(clientConn)
	req, err := http.ReadRequest(reader)
	if err != nil || req.Method != http.MethodConnect {
		return
	}

	// 2. Hedef sunucu adını ve portunu ayır
	host, port, err := net.SplitHostPort(req.Host)
	if err != nil {
		host = req.Host
		port = "443" // Varsayılan HTTPS portu
	}

	// 3. DNS Zehirlenmesini (DNS Hijacking) atlatmak için DoH (DNS over HTTPS) kullan
	realIP, err := resolveHostViaDoH(host)
	if err != nil {
		log.Printf("DNS çözülemedi %s: %v", host, err)
		return
	}

	targetAddr := net.JoinHostPort(realIP, port)

	serverConn, err := net.DialTimeout("tcp", targetAddr, 5*time.Second)
	if err != nil {
		log.Printf("Hedefe bağlanılamadı %s (%s): %v", host, targetAddr, err)
		return
	}
	defer serverConn.Close()

	// TCP Nagle algoritmasını kapatarak parçalanmış paketlerin işletim sistemi tarafından birleştirilmeden gönderilmesini sağla
	if tcpConn, ok := serverConn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
	}
	if tcpConn, ok := clientConn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
	}

	// 3. Tarayıcıya "Bağlantı Kuruldu" onayı ver
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// 4. Veri Aktarımı ve DPI Bypass (Bölme Stratejisi)
	done := make(chan bool, 2)

	// Client -> Server (Burada parçalama yapıyoruz)
	go func() {
		defer func() { done <- true }()

		// İlk paketi (genelde TLS Client Hello) yakala
		buffer := make([]byte, 32768)
		n, err := clientConn.Read(buffer)
		if err != nil {
			return
		}

		if n > 0 && buffer[0] == 0x16 { // Eğer paket TLS Handshake (HTTPS) ise
			// DPI Sistemlerini atlatmak için Agresif Parçalama (Aggressive Fragmentation)
			// DPI cihazları engelli siteyi (SNI bilgisini) okumak için paketleri analiz eder.
			// Paketleri çok ufak parçalar halinde (örneğin 1-2 byte) gönderirsek, DPI
			// bu parçaları tekrar birleştirmekte zorlanır veya birleştirme kapasitesini (buffer) aşar.

			// 1. Sadece TLS Record Header'ın ilk byte'ını gönder
			serverConn.Write(buffer[:1])
			time.Sleep(3 * time.Millisecond)

			// 2. Kalan kısmını küçük parçalarla (chunk) gönder
			chunkSize := 2
			for i := 1; i < n; i += chunkSize {
				end := i + chunkSize
				if end > n {
					end = n
				}
				serverConn.Write(buffer[i:end])

				// SNI genelde TLS paketinin ilk 200 byte'ı içerisindedir. Hızı aşırı yavaşlatmamak için
				// sadece bu kritik kısımlarda ufak beklemeler ekliyoruz.
				if i < 200 {
					time.Sleep(1 * time.Millisecond)
				}
			}
		} else {
			// TLS olmayan paketleri normal gönder
			serverConn.Write(buffer[:n])
		}

		// İlk paketten sonraki geri kalan akışı normal şekilde ilet
		io.Copy(serverConn, clientConn)
	}()

	// Server -> Client (Normal aktarım)
	go func() {
		io.Copy(clientConn, serverConn)
		done <- true
	}()

	<-done
}

// Cloudflare DoH JSON formatı için yapı
type DoHResponse struct {
	Answer []struct {
		Data string `json:"data"`
		Type int    `json:"type"`
	} `json:"Answer"`
}

// DNS Zehirlenmesini aşmak için DNS-over-HTTPS (DoH) Çözücü
func resolveHostViaDoH(host string) (string, error) {
	// Eğer gelen değer zaten bir IP adresi ise direkt döndür
	if net.ParseIP(host) != nil {
		return host, nil
	}

	// Cloudflare DNS üzerinden güvenli sorgu yap (İSS bunu engelleyemez/zehirleyemez)
	req, err := http.NewRequest("GET", "https://cloudflare-dns.com/dns-query?name="+host+"&type=A", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/dns-json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result DoHResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	for _, ans := range result.Answer {
		if ans.Type == 1 { // A kaydı (IPv4 Adresi)
			return ans.Data, nil
		}
	}
	return "", fmt.Errorf("IP adresi bulunamadı")
}
