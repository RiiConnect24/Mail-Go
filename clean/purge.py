#!/usr/bin/python
# -*- coding: utf-8 -*-

import json
import mysql.connector
from mysql.connector import errorcode

with open("../config/config.json", "rb") as f:
    config = json.load(f)

def mysql_connect():
    print("Connecting to MySQL ...")
    try:
        global cnx
        cnx = mysql.connector.connect(user=config["Username"], password=config["Password"],
                                      host=config["Host"],
                                      database=config["DBName"],
                                      charset='utf8mb4',
                                      use_unicode=True)
    except mysql.connector.Error as err:
        print "Invalid credentials" if err.errno == errorcode.ER_ACCESS_DENIED_ERROR else "Database does not exist" if err.errno == errorcode.ER_BAD_DB_ERROR else err

mysql_connect()

print("Purging old mail...")

cursor = cnx.cursor(dictionary=True, buffered=True)

query = "DELETE FROM `mails` WHERE `sent` != 1 AND `timestamp` < NOW() - INTERVAL 28 DAY"

cursor.execute(query)

print("Done!")