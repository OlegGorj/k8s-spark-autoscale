import time
from pyspark import SparkContext, SparkConf
from pyspark.sql import SQLContext, SparkSession
from pyspark.context import SparkConf
import os
app_name=os.environ['APP_NAME']
core_max=os.environ['CORE_MAX']
executor_cores=os.environ['EXECUTOR_CORES']
executor_mem=os.environ['EXECUTOR_MEM']
sc_conf = SparkConf()
sc_conf.setAppName(app_name)
sc_conf.set("spark.cores.max", core_max)
sc_conf.set("spark.executor.cores", executor_cores)
sc_conf.set("spark.executor.memory", executor_mem)
sc = None
sc = SparkContext(conf=sc_conf)
sqlContext = SQLContext(sc)

while True:
    time.sleep(1)

sc.stop()
