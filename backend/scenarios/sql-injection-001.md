---
id: "sql-injection-001"
title: "Shubhali login formasi"
subject: "IT / Web Security"
language: "uz"
status: "approved"
code_challenge_after_round: 3
code_language: "python"

facts:
  server.error_log: "Error: unexpected token near '--' in query"
  db.table: "users"
  login.field: "username"
  server.cpu: "94%"
  server.ram: "Bu ma'lumot mavjud emas"

rubric:
  - name: "Hujum turini aniqlash"
    max: 3
    keywords: ["SQL injection", "injeksiya"]
  - name: "Sababni tushuntirish"
    max: 4
    keywords: ["validatsiya", "sanitatsiya", "user input"]
  - name: "Yechim taklif qilish"
    max: 3
    keywords: ["prepared statement", "parametrlangan", "ORM"]

model_answer: >
  Bu SQL injection hujumi. Login formasi foydalanuvchi kiritmasini
  to'g'ridan-to'g'ri so'rovga qo'shgani uchun yuzaga keladi. Yechim —
  parametrlangan so'rovlar (prepared statements) ishlatish.

code_challenge:
  buggy_code: |
    def login(username, password):
        query = "SELECT * FROM users WHERE name='" + username + "' AND pass='" + password + "'"
        return db.execute(query)
  hint: "Foydalanuvchi inputini string concat bilan queryga qo'shmang."
  tests: |
    assert login("admin", "pass") is not None, "Oddiy login ishlashi kerak"
    assert login("admin' OR '1'='1", "x") is None, "SQL injection oldini olinmagan!"
    assert login("'; DROP TABLE users;--", "x") is None, "Xavfli kiritma qabul qilindi!"
---

## Vaziyat

Sen DevOps muhandisisan. Tungi 02:00 da `users` jadvaliga g'alati
so'rovlar tushayotgani haqida ogohlantirish keldi. Login sahifasida
nimadir noto'g'ri. Muammoni aniqla va hal qil.

(AI bu vaziyatni boshqaradi, lekin undan chetga chiqmaydi. Aniq raqam
so'ralsa — faqat facts ichidan get_fact orqali beradi.)
