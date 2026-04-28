# DPI Bypass macOS Proxy
# Öğrenim amacıyla yapılmıştır.
## 1. Projenin Amacı
Bu proje, İnternet Servis Sağlayıcıları (İSS) tarafından uygulanan Derin Paket İnceleme (Deep Packet Inspection - DPI) ve DNS Zehirlenmesi (DNS Hijacking) yöntemlerini aşmak amacıyla geliştirilmiş yerel bir proxy sunucu uygulamasıdır. Temel hedefi, kullanıcıların standart internet erişimlerini kısıtlayan ağ engellerini atlatmaktır. Bu projede sadece macOS'de çalışıcak şekilde kodlanmıştır. Multiplatform değildir ve sadece macos için derlenmiştir.Ama bu uygulama teknik olarak sadece bir proxy sunucudur. Yani proxy ayarlarını manuel olarak yaparsanız da çalışır size ekstra olarak windows için derlemek veya go ile çalıştırma görevide düşer ama isterseniz windows ve linux için de derleyebilirim. 

## 2. Çalışma Mantığı ve Olası Etkileri
Uygulama, çalıştırıldığı andan itibaren arka planda "127.0.0.1:8080" adresi üzerinden bir proxy başlatır ve macOS işletim sisteminin ağ ayarlarındaki (Wi-Fi ve Ethernet) proxy sunucu yapılandırmasını otomatik olarak bu adrese yönlendirir.

Ağ trafiğini yönetirken iki temel teknik kullanır:
*   **DNS-over-HTTPS (DoH):** Hedef sunucunun IP adresini standart DNS protokolü yerine Cloudflare DoH (HTTPS) üzerinden şifreli olarak çözer. Bu işlem, İSS'lerin sahte IP adresleri döndürerek (DNS Zehirlenmesi) erişimi engelleme durumunu atlatmaya çalışır ama ip adresi bir şekilde engellenirse bu atlatmalar malesef çalışmaz.
*   **Agresif Parçalama (Aggressive Fragmentation):** Güvenli bağlantı kurulum (TLS ClientHello) evresinde iletilen Server Name Indication (SNI) verisi, DPI cihazlarının analiz kapasitesini zorlamak amacıyla küçük (2 byte boyutunda) TCP paketlerine bölünerek hedefe iletilir. Paketlerin işletim sistemi tarafından birleştirilmeden gönderilmesini garanti altına almak için TCP Nagle algoritması devre dışı bırakılır. Normalde Windows içindeki winenet api paketleri birleştirir ama biz bu projede Go'nın socket api'sini kullanıyoruz. Bu yüzden Windows'daki gibi bir 3. katman kontrolüne sahip değiliz ama bu şekilde macos ve linux işletim sistemlerinde de çalışan bir uygulamaya sahip olmuş oluyoruz.

**Kullanıcılara Yaratabileceği Etkiler ve Olası Sorunlar:**
Uygulama aktif durumdayken cihazın tüm HTTP ve HTTPS bağlantıları bu yerel vekil sunucu üzerinden yönlendirilir. Yüksek bant genişliği gerektiren işlemler veya proxy sunucu kullanımını desteklemeyen spesifik kurumsal uygulamalarda minimal gecikmeler gözlemlenebilir. Uygulamanın kendi menüsü üzerinden usulüne uygun şekilde kapatılmaması **(işletim sistemi tarafından zorla sonlandırılması veya çökmesi)** durumunda, macOS ağ ayarlarındaki proxy sunucu yönlendirmesi açık kalabilir. Bu senaryoda internet erişimi kesilecektir. Sorunun **çözümü için** uygulamanın tekrar başlatılıp menüsü üzerinden kapatılması **veya** macOS **"Sistem Ayarları > Ağ > Proxy"** sekmesindeki ayarların manuel olarak **kapatılması** gerekir.

## 3. Kullanım, Kapatma ve Silme Talimatları

**Nasıl Kullanılır?**
1. Uygulamayı GitHub dosyaları arasından bilgisayarınıza indirin.
2. İndirdiğiniz uygulamaya çift tıklayarak açın. (macOS güvenlik politikalarından dolayı ilk açılışta dosyaya sağ tıklayıp "Aç" seçeneğini seçmeniz gerekebilir).
3. Açıldıktan sonra uygulama sağ üstte menü çubuğunda görünür duruma gelecektir; bu durum uygulamanın aktif ve çalışıyor olduğu anlamına gelir.

**Nasıl Kapatılır?**
Kapatmak için menü çubuğunda bulunan "DPI Bypass" uygulama yazısına tıklayın ve alt kısımda bulunan "Kapat" seçeneğine tıklayarak uygulamayı sonlandırın.

**Nasıl Silinir?**
Silmek için doğrudan uygulama kapatıldıktan sonra dosyayı silin. Uygulama bilgisayarınızda herhangi bir kalıcı arka plan hizmeti veya veri oluşturmaz.
