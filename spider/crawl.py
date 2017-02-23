# -*- coding=utf8 -*-
"""从知乎上爬取指定话题的精华回答"""
import os
import re
import time
from collections import namedtuple

import requests
from requests.exceptions import RequestException
from pyquery import PyQuery
from sqlalchemy.orm import sessionmaker
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy import (
    Column,
    String,
    Text,
    Integer,
    create_engine
)


session = requests.Session()
BaseModel = declarative_base()
headers = {
    "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
    "Accept-Encoding": "gzip, deflate, sdch, br",
    "Accept-Language": "zh-CN,zh;q=0.8,en;q=0.6,zh-TW;q=0.4",
    "Cache-Control": "max-age=0",
    "Connection": "keep-alive",
    "Host": "www.zhihu.com",
    "Upgrade-Insecure-Requests": "1",
    "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36",
}

# 需要爬取的话题精华
topic_list = ('https://www.zhihu.com/topic/19564408/top-answers',     # 爱情
              'https://www.zhihu.com/topic/19553155/top-answers',)    # 婚姻

# 回答结构体 字段含义: 问题内容 问题标签 回复内容 点赞数 评论数
Answer = namedtuple('Answer', 'labels question answer star')

# 当前路径以及数据库路径
current_dir = os.path.dirname(os.path.abspath(__file__))
sql_path = os.path.join(current_dir, 'tables.sql')
db_path = os.path.join(current_dir, 'tables.sqlite')

# 数据库引擎和session
engine = create_engine('sqlite:///{}'.format(db_path))
DBSession = sessionmaker(bind=engine)


def get_page(url):
    """爬取url数据

    Args:
        url (string): 网址链接

    Returns:
        unicode 网页内容
    """
    try:
        response = session.get(url, headers=headers, timeout=10)
    except RequestException as e:
        print '{} error {}'.format(url, e)
    else:
        if response.ok:
            return response.text


def parse_answer_url(page):
    """解析回答链接

    Args:
        page (unicode): 精华回答列表页面内容

    Returns:
        list<string>: 精华回答url列表
    """
    result = []
    if not page:
        print 'invalid content'
        return result

    html = PyQuery(page)
    for item in html("div[class='expandable entry-body']>link"):
        result.append('https://www.zhihu.com{}'.format(item.get('href')))
    return result


def parse_answer_page(page):
    """解析回答详情页

    Args:
        page (unicode): 回答详情页内容

    Returns:
        Answer 回答详情
    """
    if not page:
        print 'invalid content'
        return None

    html = PyQuery(page)
    labels = []
    for item in html("div[class='zm-tag-editor-labels zg-clear']>a"):
        labels.append(item.text.strip())

    question = html("div[id='zh-question-title']>h2[class='zm-item-title']>a")[0].text_content().strip()
    star = int(html("span[class='js-voteCount']")[0].text)
    answer = html("div[class='zm-editable-content clearfix']")[0].text_content().strip()

    return Answer(labels=labels, question=question, answer=answer, star=star)


def parse_answer_id(url):
    """根据url解析出问题id和回答id

    Args:
        url (string): 回答详情页url

    Returns:
        tuple 问题id和回答id
    """
    question_id, answer_id = re.findall(r'\d+', url)
    return int(question_id), int(answer_id)


def init_tables():
    """创建表"""
    if os.path.exists(db_path):
        os.remove(db_path)

    with open(db_path, 'w') as f:
        pass

    with open(sql_path, 'r') as f:
        sx = DBSession()
        for sql in f.read().split(';'):
            sx.execute(sql)
        sx.commit()


def add_answer(question_id, answer_id, answer):
    """向数据库中添加数据

    Args:
        question_id (int): 问题id
        answer_id (int): 回答id
        answer (Answer): 回答详情
    """
    if not answer:
        return

    exist_question = AnswerTable.exist_question(question_id)
    sx = DBSession()
    answer_record = AnswerTable(
        question_id=question_id,
        answer_id=answer_id,
        question=answer.question,
        answer=answer.answer,
        star=answer.star,
    )
    sx.add(answer_record)

    if answer.labels and not exist_question:
        label_records = [LabelTable(question_id=question_id, label=x)
                         for x in answer.labels]
        sx.add_all(label_records)
    sx.commit()


class AnswerTable(BaseModel):

    __tablename__ = 'answer'

    id = Column(Integer, primary_key=True)
    question_id = Column(Integer, nullable=False)
    answer_id = Column(Integer, nullable=False)
    question = Column(Text, default=u'')
    answer = Column(Text, default=u'')
    star = Column(Integer, nullable=False)

    @classmethod
    def exist_question(cls, question_id):
        """问题是否已经添加

        Args:
            question_id (int): 问题id
        """
        sx = DBSession()
        result = sx.query(cls.id).filter(cls.question_id == question_id).first()
        return True if result else None

    @classmethod
    def exist_answer(cls, question_id, answer_id):
        """回答是否已经添加

        Args:
            question_id (int): 问题id
            answer_id (int): 回答id
        """
        sx = DBSession()
        result = sx.query(cls.id).filter(cls.question_id == question_id).\
            filter(cls.answer_id == answer_id).first()
        return True if result else None


class LabelTable(BaseModel):

    __tablename__ = 'label'

    id = Column(Integer, primary_key=True)
    question_id = Column(Integer, nullable=False)
    label = Column(String(50), default=u'')


if __name__ == '__main__':
    init_tables()
    for topic_url in topic_list:
        for page in xrange(1, 51):
            ans_list_url = '{}?page={}'.format(topic_url, page)
            print ans_list_url

            answer_list_page = get_page(ans_list_url)
            answer_urls = parse_answer_url(answer_list_page)

            for ans_url in answer_urls:
                print ans_url
                question_id, answer_id = parse_answer_id(ans_url)
                if AnswerTable.exist_answer(question_id, answer_id):
                    continue

                answer_page = get_page(ans_url)
                answer = parse_answer_page(answer_page)
                add_answer(question_id, answer_id, answer)

                time.sleep(1)
            time.sleep(10)
        time.sleep(120)
