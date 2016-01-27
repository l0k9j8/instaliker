#!/usr/bin/env python
# -*- coding: utf-8 -*-

from urllib2 import Request, build_opener, HTTPCookieProcessor, HTTPHandler
from urllib import urlencode
import cookielib
import re
from json import loads

LOGIN = 'l0k9j8'
PASSWORD = '********'

INSTAGRAM_HOST = 'https://www.instagram.com'
AUTH_URI = '/accounts/login/ajax/'
LIKE_URI = '/web/likes/%s/like/'

HEADER = {'x-requested-with': 'XMLHttpRequest',
          'x-instagram-ajax': 1,
          'user-agent': 'Mozilla/5.0',
          'authority': 'www.instagram.com',
          'referer': 'https://www.instagram.com/'}


class ParseError(Exception):
    pass


class CoreError(Exception):
    pass


def csrf_token_from_cookie(cookies):
    for cookie in cookies:
        if cookie.name == 'csrftoken':
            return cookie.value
    return ''


def loader(opener, url, data=None, headers=None):
    req = Request(url, data=data, headers=headers)
    return opener.open(req).read()


def get_feed_page(user_feed):
    if 'entry_data' not in user_feed:
        raise ParseError('Entry data not found')
    if 'FeedPage' not in user_feed['entry_data']:
        raise ParseError('FeedPage not found')
    return user_feed['entry_data']['FeedPage']


def main():
    cookies = cookielib.CookieJar()
    opener = build_opener(HTTPCookieProcessor(cookies), HTTPHandler())
    headers = HEADER.copy()
    loader(opener, INSTAGRAM_HOST, headers=headers)
    headers['x-csrftoken'] = csrf_token_from_cookie(cookies)
    auth_data = loads(loader(opener, INSTAGRAM_HOST + AUTH_URI, data=urlencode({'username': LOGIN,
                                                                                'password': PASSWORD}),
                             headers=headers))

    if not auth_data['authenticated']:
        raise CoreError('Bad auth')
    feed_regexp = re.compile(r'window\._sharedData = (.+);</script>')
    feed_json_strings = feed_regexp.findall(loader(opener, INSTAGRAM_HOST, headers=headers))
    if feed_json_strings:
        user_feed_data = loads(feed_json_strings[0])
    else:
        raise CoreError('Bad regexp')

    headers['x-csrftoken'] = csrf_token_from_cookie(cookies)

    for feed in get_feed_page(user_feed_data):
        if 'feed' in feed and 'media' in feed['feed']:
            nodes = feed['feed']['media']['nodes']
        else:
            continue

        for node in nodes:
            print 'Image url: %s, Has liked: %s' % (node['display_src'], node['likes']['viewer_has_liked'])
            if not node['likes']['viewer_has_liked']:
                loads(loader(opener, INSTAGRAM_HOST + LIKE_URI % node['id'], headers=headers, data=''))
                print 'Like image with id: %s' % node['id']


if __name__ == '__main__':
    main()
