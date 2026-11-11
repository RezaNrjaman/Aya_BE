# Revisi Modul Manajemen — Dropdown, Profil Unit, dan Hak Akses

## Perubahan
1. Nomor urutan di label sidebar Manajemen dihapus.
2. Form dan tabel Unit Usaha hanya menampilkan nama dan deskripsi (status dan aksi tetap tersedia sebagai kontrol operasional). Kolom kontak/alamat lama di database tidak dihapus agar data lama tetap aman.
3. Semua dropdown organisasi menggunakan `SearchableSelect` dengan pencarian langsung.
4. Role dan Pengguna memuat Unit Usaha, Kategori, dan Divisi secara berhalaman (maksimal 100 item per request) hingga seluruh halaman selesai. Ini memperbaiki bug lama: frontend meminta 200–1000 item padahal backend menolak `per_page` di atas 100, sehingga request gagal dan dropdown menjadi kosong.
5. Kategori disaring berdasarkan `unit_usaha_id`; Divisi disaring berdasarkan unit dan kategori; Role disaring berdasarkan divisi. Mengganti induk mengosongkan pilihan anak.
6. Pembuatan role mewajibkan minimal satu permission eksplisit. Opsi/placeholder “Mulai tanpa hak akses” dihapus; permission dipilih dari katalog yang boleh didelegasikan akun Partner. Backend juga menolak role Partner dengan permission kosong.

## Kesesuaian dengan schema Supabase terbaru
Schema yang diberikan sudah memiliki kolom relasi yang dipakai: `business_categories.unit_usaha_id`, `divisions.unit_usaha_id`, `divisions.category_id`, `roles.division_id`, serta `user_role_assignments.role_id` dan `unit_usaha_id`. `users` menyimpan `division_id` dan role pengguna tetap disimpan pada `user_role_assignments`.

Tidak ada migration database baru untuk revisi ini. Kolom kontak/alamat pada `business_units` dibiarkan sebagai kolom legacy/opsional dan tidak lagi diminta oleh UI.

## Verifikasi
- Frontend TypeScript check: lulus (`tsc --noEmit -p tsconfig.app.json`).
- Go formatting: lulus (`gofmt`).
- Go unit/integration test belum dapat dijalankan karena dependency belum tersedia lokal dan akses `proxy.golang.org` tidak tersedia pada lingkungan pemeriksaan.
- Vite production build perlu dijalankan di mesin pengembangan setelah dependency di-install ulang.
