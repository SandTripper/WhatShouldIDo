import pymysql
import json

if __name__ == "__main__":
    db = pymysql.connect(host='localhost',
                         user='root',
                         password='',
                         database='what_should_i_do',
                         charset='utf8mb4')
    cursor = db.cursor()

    datas = []
    with open("datas.json", "r", encoding="utf-8") as f:
        datas = json.load(f)

    insert_str = "INSERT INTO job_information_tb values(null,%s,%s,%s)"
    try:
        cursor.execute("set names utf8mb4")
        for dic in datas:
            cursor.execute(
                insert_str, (dic["recruitment_unit"], dic["post_name"], dic["require_text"]))
        db.commit()
    except:
        db.rollback()

    db.close()
    cursor.close()
