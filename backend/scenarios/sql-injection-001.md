---
id: "econ-breakeven-001"
title: "Bozorda narxning keskin tushishi (Demping)"
subject: "Iqtisodiyot / Mikroiqtisodiyot"
language: "uz"
status: "approved"
code_challenge_after_round: 3
code_language: "python"

facts:
  company.fixed_costs: "10000 USD"
  company.variable_cost_per_unit: "5 USD"
  market.old_price: "15 USD"
  market.new_price: "8 USD"
  competitor.strategy: "Demping (arzonlashtirish)"

rubric:
  - name: "Zararsizlik nuqtasini aniqlash"
    max: 3
    keywords: ["zararsizlik nuqtasi", "break-even", "qoplash"]
  - name: "Bozor holatini tahlil qilish"
    max: 4
    keywords: ["demping", "marja", "foyda", "zarar"]
  - name: "Strategik yechim taklif qilish"
    max: 3
    keywords: ["xarajatni kamaytirish", "narx strategiyasi", "ishlab chiqarish hajmi", "optimallashtirish"]

model_answer: >
  Bozorda raqobatchi tomonidan demping (narxni sun'iy pasaytirish) qilinmoqda.
  Oldingi narxda (15$) zararsizlik nuqtasi 1000 dona mahsulot edi. Yangi narxda
  (8$) bir mahsulotdan tushadigan foyda marjasi 3$ gacha tushib ketdi, natijada 
  zararsizlik nuqtasi 3334 donagacha oshdi. Yechim — o'zgaruvchan xarajatlarni
  optimallashtirish yoki vaqtincha sotuv hajmini oshirish strategiyasini qo'llash.

code_challenge:
  buggy_code: |
    def calculate_break_even(fixed_costs, variable_cost, selling_price):
        # Xato: Maxrajda noto'g'ri arifmetik amal bajarilgan
        margin = selling_price + variable_cost
        return fixed_costs / margin
  hint: "Zararsizlik nuqtasi (Break-even Point) = Doimiy xarajatlar / (Sotuv narxi - O'zgaruvchan xarajat)."
  tests: |
    assert calculate_break_even(10000, 5, 15) == 1000, "15$ sotuv narxida 1000 dona bo'lishi kerak"
    assert calculate_break_even(12000, 4, 10) == 2000, "10$ sotuv narxida 2000 dona bo'lishi kerak"
---

## Vaziyat

Sen yirik ishlab chiqarish kompaniyasining bosh iqtisodchisisan. Juma kuni kechqurun bozorda raqobatchilar o'z mahsulotlari narxini keskin tushirib yuborganligi haqida xabar keldi. Kompaniya rahbari sendan ishlab chiqarishni shu yangi narxda davom ettirish moliyaviy jihatdan qanchalik xavfsiz ekanligi haqida tezkor xulosa so'ramoqda. Muammoni tahlil qil, xavfni bahola va tegishli qaror taklif qil.

(AI bu vaziyatni boshqaradi, lekin undan chetga chiqmaydi. Aniq raqam
so'ralsa — faqat facts ichidan get_fact orqali beradi.)