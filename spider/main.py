from selenium import webdriver
import time
import json
import threading
from selenium.webdriver.common.action_chains import ActionChains
from selenium.webdriver.common.keys import Keys


datas = []
mutex = threading.Lock()


def click_next_page(driver, next_page_key, find_next_page_by_class):
    js = "var q=document.documentElement.scrollTop=100000"
    driver.execute_script(js)
    time.sleep(1)
    if find_next_page_by_class:
        driver.find_element_by_class_name(next_page_key).click()
    else:
        driver.find_element_by_xpath(next_page_key).click()
    time.sleep(3)
    js = "var q=document.documentElement.scrollTop=0"
    driver.execute_script(js)
    time.sleep(1)


def run(recruitment_unit, url, page_from, page_to, find_next_page_by_class, next_page_key, box_xpath, box_from, box_to, name_xml, require_xml):
    global datas
    global mutex

    options = webdriver.ChromeOptions()

    options.add_argument('-ignore-certificate-errors')
    options.add_argument('-ignore -ssl-errors')

    driver = webdriver.Chrome("chromedriver.exe", chrome_options=options)

    driver.implicitly_wait(time_to_wait=5)

    driver.get(url)
    time.sleep(3)

    for page_index in range(page_from-1):
        click_next_page(
            driver=driver, next_page_key=next_page_key, find_next_page_by_class=find_next_page_by_class)

    for page_index in range(page_from, page_to+1):

        print("{}: getting page {} ".format(recruitment_unit, page_index))

        main_handle = driver.current_window_handle
        for i in range(box_from, box_to+1):
            last_handle = driver.current_window_handle
            try:
                ActionChains(driver).key_down(Keys.CONTROL).perform()
                driver.find_element_by_xpath(box_xpath.format(i)).click()
                ActionChains(driver).key_up(Keys.CONTROL).perform()
                if i == -1:
                    js = "var q=document.documentElement.scrollTop=100000"
                    driver.execute_script(js)
                    time.sleep(1)
            except:
                print(recruitment_unit, "get box error")
            driver.switch_to_window(last_handle)
            time.sleep(1)

        time.sleep(1)
        all_handles = driver.window_handles
        for handle in all_handles:
            if main_handle == handle:
                continue
            driver.switch_to_window(handle)
            try:
                post_name = driver.find_element_by_xpath(name_xml).text
                require_text = driver.find_element_by_xpath(require_xml).text
                data = {
                    "recruitment_unit": recruitment_unit,
                    "post_name": post_name,
                    "require_text": require_text
                }
                mutex.acquire()
                datas.append(data)
                mutex.release()

            except:
                print("getting 岗位信息 error")

            driver.close()

        driver.switch_to_window(main_handle)
        click_next_page(
            driver=driver, next_page_key=next_page_key, find_next_page_by_class=find_next_page_by_class)

    driver.close()
    driver.quit()


if __name__ == '__main__':
    hooks = {}
    with open('spider_hook.json', "r", encoding="utf-8")as f:
        hooks = json.load(f)

    with open('datas.json', "r", encoding="utf-8")as f:
        datas = json.load(f)

    threads = []
    for key in hooks:
        value = hooks[key]
        if not value["enable"]:
            continue
        t = threading.Thread(target=run, args=(key, value["url"], value["page_from"], value["page_to"], value["find_next_page_by_class"] if "find_next_page_by_class" in value else False, value["next_page_key"],
                             value["box_xpath"], value["box_from"], value["box_to"], value["name_xml"], value["require_xml"]))
        t.start()
        threads.append(t)
    for t in threads:
        t.join()

    with open("datas.json", "w", encoding='utf8') as f:
        json.dump(datas, f, ensure_ascii=False)
